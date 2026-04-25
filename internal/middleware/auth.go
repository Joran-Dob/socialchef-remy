package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/socialchef/remy/internal/config"
)

type contextKey string

const UserIDKey contextKey = "userID"

type JWKSKey struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n,omitempty"`
	E   string `json:"e,omitempty"`
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type JWKSResponse struct {
	Keys []JWKSKey `json:"keys"`
}

type JWKSManager struct {
	url             string
	apiKey          string
	rsaKeys         map[string]*rsa.PublicKey
	ecKeys          map[string]*ecdsa.PublicKey
	mu              sync.RWMutex
	expiresAt       time.Time
	refreshInterval time.Duration
}

func NewJWKSManager(supabaseURL, apiKey string) *JWKSManager {
	return &JWKSManager{
		url:             supabaseURL + "/auth/v1/.well-known/jwks.json",
		apiKey:          apiKey,
		rsaKeys:         make(map[string]*rsa.PublicKey),
		ecKeys:          make(map[string]*ecdsa.PublicKey),
		refreshInterval: 1 * time.Hour,
	}
}

func (j *JWKSManager) GetRSAKey(kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	key, exists := j.rsaKeys[kid]
	expired := time.Now().After(j.expiresAt)
	j.mu.RUnlock()

	if exists && !expired {
		return key, nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if key, exists := j.rsaKeys[kid]; exists && time.Now().Before(j.expiresAt) {
		return key, nil
	}

	if err := j.refresh(); err != nil {
		if key, exists := j.rsaKeys[kid]; exists {
			return key, nil
		}
		return nil, err
	}

	key, exists = j.rsaKeys[kid]
	if !exists {
		return nil, fmt.Errorf("RSA key with kid %s not found", kid)
	}

	return key, nil
}

func (j *JWKSManager) GetECKey(kid string) (*ecdsa.PublicKey, error) {
	j.mu.RLock()
	key, exists := j.ecKeys[kid]
	expired := time.Now().After(j.expiresAt)
	j.mu.RUnlock()

	if exists && !expired {
		return key, nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if key, exists := j.ecKeys[kid]; exists && time.Now().Before(j.expiresAt) {
		return key, nil
	}

	if err := j.refresh(); err != nil {
		if key, exists := j.ecKeys[kid]; exists {
			return key, nil
		}
		return nil, err
	}

	key, exists = j.ecKeys[kid]
	if !exists {
		return nil, fmt.Errorf("EC key with kid %s not found", kid)
	}

	return key, nil
}

func (j *JWKSManager) refresh() error {
	req, err := http.NewRequest("GET", j.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if j.apiKey != "" {
		req.Header.Set("apikey", j.apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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

	newRSAKeys := make(map[string]*rsa.PublicKey)
	newECKeys := make(map[string]*ecdsa.PublicKey)

	for _, key := range jwks.Keys {
		if key.Use != "" && key.Use != "sig" {
			continue
		}

		switch key.Kty {
		case "RSA":
			pubKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				continue
			}
			newRSAKeys[key.Kid] = pubKey

		case "EC":
			pubKey, err := parseECPublicKey(key.Crv, key.X, key.Y)
			if err != nil {
				continue
			}
			newECKeys[key.Kid] = pubKey
		}
	}

	j.rsaKeys = newRSAKeys
	j.ecKeys = newECKeys
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

func parseECPublicKey(crv, xStr, yStr string) (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve: %s", crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// uuidRegex matches a canonical 8-4-4-4-12 lowercase-or-uppercase UUID. We
// only allow the X-On-Behalf-Of header to impersonate a valid-looking user
// uuid so we don't write arbitrary strings into Supabase user_id columns.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	var jwksManager *JWKSManager
	if cfg.SupabaseURL != "" {
		jwksManager = NewJWKSManager(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sibling services (e.g. Sous) may send the shared secret in `X-API-Key`
			// instead of `Authorization: Bearer` so a hex string is not mistaken for
			// a JWT. Same rules as the Bearer service-token path: requires
			// `X-On-Behalf-Of: <user uuid>`.
			if cfg.InternalServiceToken != "" {
				xKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
				if xKey != "" {
					if subtle.ConstantTimeCompare([]byte(xKey), []byte(cfg.InternalServiceToken)) != 1 {
						http.Error(w, "Unauthorized: invalid X-API-Key", http.StatusUnauthorized)
						return
					}
					onBehalfOf := strings.TrimSpace(r.Header.Get("X-On-Behalf-Of"))
					if onBehalfOf == "" {
						http.Error(w, "Unauthorized: service token requires X-On-Behalf-Of header", http.StatusUnauthorized)
						return
					}
					if !uuidRegex.MatchString(onBehalfOf) {
						http.Error(w, "Unauthorized: X-On-Behalf-Of must be a user UUID", http.StatusUnauthorized)
						return
					}
					ctx := context.WithValue(r.Context(), UserIDKey, onBehalfOf)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

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

			// Service-token auth path: trusted sibling services (e.g. Sous)
			// present the shared secret plus `X-On-Behalf-Of: <uuid>` to
			// impersonate the target user. We short-circuit JWT parsing.
			if cfg.InternalServiceToken != "" && subtle.ConstantTimeCompare([]byte(tokenString), []byte(cfg.InternalServiceToken)) == 1 {
				onBehalfOf := strings.TrimSpace(r.Header.Get("X-On-Behalf-Of"))
				if onBehalfOf == "" {
					http.Error(w, "Unauthorized: service token requires X-On-Behalf-Of header", http.StatusUnauthorized)
					return
				}
				if !uuidRegex.MatchString(onBehalfOf) {
					http.Error(w, "Unauthorized: X-On-Behalf-Of must be a user UUID", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, onBehalfOf)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				switch token.Method.(type) {
				case *jwt.SigningMethodRSA:
					if jwksManager == nil {
						return nil, fmt.Errorf("JWKS not configured")
					}

					kid, ok := token.Header["kid"].(string)
					if !ok {
						return nil, fmt.Errorf("RSA token missing kid header")
					}

					return jwksManager.GetRSAKey(kid)

				case *jwt.SigningMethodECDSA:
					if jwksManager == nil {
						return nil, fmt.Errorf("JWKS not configured")
					}

					kid, ok := token.Header["kid"].(string)
					if !ok {
						return nil, fmt.Errorf("ECDSA token missing kid header")
					}

					return jwksManager.GetECKey(kid)

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
