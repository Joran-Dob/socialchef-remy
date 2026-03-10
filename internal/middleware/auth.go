package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/socialchef/remy/internal/config"
)

type contextKey string

const UserIDKey contextKey = "userID"

type JWKSManager struct {
	url             string
	keys            map[string]*rsa.PublicKey
	mu              sync.RWMutex
	expiresAt       time.Time
	refreshInterval time.Duration
}

type JWKSResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func NewJWKSManager(supabaseURL string) *JWKSManager {
	return &JWKSManager{
		url:             supabaseURL + "/auth/v1/jwks",
		keys:            make(map[string]*rsa.PublicKey),
		refreshInterval: 1 * time.Hour,
	}
}

func (j *JWKSManager) GetKey(kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	key, exists := j.keys[kid]
	expired := time.Now().After(j.expiresAt)
	j.mu.RUnlock()

	if exists && !expired {
		return key, nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if key, exists := j.keys[kid]; exists && time.Now().Before(j.expiresAt) {
		return key, nil
	}

	if err := j.refresh(); err != nil {
		if key, exists := j.keys[kid]; exists {
			return key, nil
		}
		return nil, err
	}

	key, exists = j.keys[kid]
	if !exists {
		return nil, fmt.Errorf("key with kid %s not found", kid)
	}

	return key, nil
}

func (j *JWKSManager) refresh() error {
	resp, err := http.Get(j.url)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Use != "" && key.Use != "sig" {
			continue
		}

		pubKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			continue
		}

		newKeys[key.Kid] = pubKey
	}

	j.keys = newKeys
	j.expiresAt = time.Now().Add(j.refreshInterval)
	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	var jwksManager *JWKSManager
	if cfg.SupabaseURL != "" {
		jwksManager = NewJWKSManager(cfg.SupabaseURL)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				switch token.Method.(type) {
				case *jwt.SigningMethodRSA:
					if jwksManager == nil {
						return nil, fmt.Errorf("JWKS not configured")
					}

					kid, ok := token.Header["kid"].(string)
					if !ok {
						return nil, fmt.Errorf("RS256 token missing kid header")
					}

					return jwksManager.GetKey(kid)

				case *jwt.SigningMethodHMAC:
					if cfg.SupabaseJWTSecret == "" {
						return nil, fmt.Errorf("JWT secret not configured")
					}
					return []byte(cfg.SupabaseJWTSecret), nil

				default:
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
			})

			if err != nil || !token.Valid {
				http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Unauthorized: Invalid claims", http.StatusUnauthorized)
				return
			}

			iss, _ := claims["iss"].(string)
			expectedIss := cfg.SupabaseURL + "/auth/v1"
			if iss != expectedIss {
				http.Error(w, "Unauthorized: Invalid issuer", http.StatusUnauthorized)
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				http.Error(w, "Unauthorized: Missing sub claim", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := GetUserID(r.Context()); !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
