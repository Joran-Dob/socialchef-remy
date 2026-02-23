package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/socialchef/remy/internal/config"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	supabaseURL := "https://test.supabase.co"
	cfg := &config.Config{
		SupabaseURL:       supabaseURL,
		SupabaseJWTSecret: secret,
	}

	createToken := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(secret))
		return tokenString
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID string
	}{
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Authorization header format",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token format",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Expired token",
			authHeader: "Bearer " + createToken(jwt.MapClaims{
				"sub": "user-123",
				"iss": supabaseURL + "/auth/v1",
				"exp": time.Now().Add(-time.Hour).Unix(),
			}),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid signature",
			authHeader: "Bearer " + func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "user-123",
					"iss": supabaseURL + "/auth/v1",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				return tokenString
			}(),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid issuer",
			authHeader: "Bearer " + createToken(jwt.MapClaims{
				"sub": "user-123",
				"iss": "https://wrong.supabase.co/auth/v1",
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Valid token",
			authHeader: "Bearer " + createToken(jwt.MapClaims{
				"sub": "user-123",
				"iss": supabaseURL + "/auth/v1",
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := GetUserID(r.Context())
				if !ok {
					t.Error("expected userID in context")
				}
				if userID != tt.expectedUserID {
					t.Errorf("expected userID %s, got %s", tt.expectedUserID, userID)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	t.Run("No user in context", func(t *testing.T) {
		handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}
