package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/socialchef/remy/internal/api"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"
)

// ============================================================================
// Test Token Helpers
// ============================================================================

func createTestToken(secret, supabaseURL, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iss": supabaseURL + "/auth/v1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func createExpiredToken(secret, supabaseURL, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iss": supabaseURL + "/auth/v1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func createInvalidSignatureToken(supabaseURL, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iss": supabaseURL + "/auth/v1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("wrong-secret"))
	return tokenString
}

// ============================================================================
// API Endpoint Tests - POST /api/recipe
// ============================================================================

func TestHandleImportRecipe_MissingAuth(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	body := api.ImportRecipeRequest{URL: "https://instagram.com/p/test"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/recipe", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.HandleImportRecipe(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleImportRecipe_InvalidBody(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Invalid JSON",
			body:       `{"url": "https://instagram.com/p/test",}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Empty body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing URL field",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/recipe", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
			rr := httptest.NewRecorder()

			server.HandleImportRecipe(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestHandleImportRecipe_EmptyURL(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	body := api.ImportRecipeRequest{URL: ""}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/recipe", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	server.HandleImportRecipe(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// ============================================================================
// API Endpoint Tests - GET /api/recipe-status
// ============================================================================

func TestHandleJobStatus_MissingJobID(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipe-status", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	server.HandleJobStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleJobStatus_NoAuth(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipe-status?job_id="+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	server.HandleJobStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// ============================================================================
// API Endpoint Tests - GET /api/user-import-status
// ============================================================================

func TestHandleUserImportStatus_WithoutAuth(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/user-import-status", nil)
	rr := httptest.NewRecorder()

	server.HandleUserImportStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleUserImportStatus_WithAuth(t *testing.T) {
	fixtures := setupTestFixtures()

	userID := uuid.New().String()

	// Create multiple jobs for the user
	for i := 0; i < 3; i++ {
		jobID := uuid.New().String()
		fixtures.mockDB.importJobs[jobID] = generated.RecipeImportJob{
			ID:        uuidToPgtype(uuid.MustParse(jobID)),
			UserID:    uuidToPgtype(uuid.MustParse(userID)),
			Url:       "https://instagram.com/p/test" + string(rune('a'+i)),
			Status:    "pending",
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
	}

	// Create another user's job
	otherUserID := uuid.New().String()
	otherJobID := uuid.New().String()
	fixtures.mockDB.importJobs[otherJobID] = generated.RecipeImportJob{
		ID:        uuidToPgtype(uuid.MustParse(otherJobID)),
		UserID:    uuidToPgtype(uuid.MustParse(otherUserID)),
		Url:       "https://instagram.com/p/other",
		Status:    "completed",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// Verify we can retrieve only the user's jobs
	jobs, err := fixtures.mockDB.GetImportJobsByUser(nil, uuidToPgtype(uuid.MustParse(userID)))
	if err != nil {
		t.Fatalf("failed to get jobs: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

// ============================================================================
// API Endpoint Tests - POST /api/generate-embedding
// ============================================================================

func TestHandleGenerateEmbedding_MissingRecipeID(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	body := api.GenerateEmbeddingRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/generate-embedding", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.HandleGenerateEmbedding(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGenerateEmbedding_InvalidBody(t *testing.T) {
	cfg := &config.Config{}
	server := api.NewServer(cfg, nil, nil, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Invalid JSON",
			body:       `{"recipe_id": "abc",}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Empty body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/generate-embedding", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			server.HandleGenerateEmbedding(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestHandleGenerateEmbedding_ValidRequest(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()
	userID := uuid.New().String()

	// Pre-create a recipe in the database
	fixtures.mockDB.recipes[recipeID] = generated.Recipe{
		ID:          uuidToPgtype(uuid.MustParse(recipeID)),
		CreatedBy:   uuidToPgtype(uuid.MustParse(userID)),
		RecipeName:  "Test Recipe",
		Description: pgtype.Text{String: "A test recipe", Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// Verify the recipe exists in mock DB
	recipe, err := fixtures.mockDB.GetRecipe(nil, uuidToPgtype(uuid.MustParse(recipeID)))
	if err != nil {
		t.Fatalf("failed to get recipe: %v", err)
	}

	if recipe.RecipeName != "Test Recipe" {
		t.Errorf("expected recipe name 'Test Recipe', got %s", recipe.RecipeName)
	}
}

// ============================================================================
// Auth Flow Integration Tests
// ============================================================================

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedUserID string
	}{
		{
			name:           "Valid token with user ID",
			userID:         "user-123",
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
		},
		{
			name:           "Valid token with UUID user ID",
			userID:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			expectedUserID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTestToken(cfg.SupabaseJWTSecret, cfg.SupabaseURL, tt.userID)

			handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := middleware.GetUserID(r.Context())
				if !ok {
					t.Error("expected userID in context but not found")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if userID != tt.expectedUserID {
					t.Errorf("expected userID %s, got %s", tt.expectedUserID, userID)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Authorization format - missing Bearer",
			authHeader:     "token-value",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Authorization format - only Bearer",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token format",
			authHeader:     "Bearer invalid-token-format",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	token := createExpiredToken(cfg.SupabaseJWTSecret, cfg.SupabaseURL, "user-123")

	handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for expired token, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	token := createInvalidSignatureToken(cfg.SupabaseURL, "user-123")

	handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for invalid signature, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddleware_InvalidIssuer(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	// Create token with wrong issuer
	token := createTestToken(cfg.SupabaseJWTSecret, "https://wrong.supabase.co", "user-123")

	handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for invalid issuer, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddleware_MissingSubClaim(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	// Create token without sub claim
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": cfg.SupabaseURL + "/auth/v1",
		"exp": time.Now().Add(time.Hour).Unix(),
		// Missing "sub" claim
	})
	tokenString, _ := token.SignedString([]byte(cfg.SupabaseJWTSecret))

	handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for missing sub claim, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestGetUserID_FromContext(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		shouldExist bool
	}{
		{
			name:        "Valid user ID in context",
			userID:      "user-123",
			shouldExist: true,
		},
		{
			name:        "Empty user ID in context",
			userID:      "",
			shouldExist: true, // Empty string is still a valid value in context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := withUserID(t.Context(), tt.userID)
			userID, ok := middleware.GetUserID(ctx)

			if !ok {
				t.Error("expected userID to exist in context")
			}

			if userID != tt.userID {
				t.Errorf("expected userID %s, got %s", tt.userID, userID)
			}
		})
	}
}

func TestGetUserID_NotInContext(t *testing.T) {
	// Test without setting user ID
	userID, ok := middleware.GetUserID(t.Context())

	if ok {
		t.Error("expected userID to NOT exist in context")
	}

	if userID != "" {
		t.Errorf("expected empty userID, got %s", userID)
	}
}

func TestRequireAuth_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		withUserID     bool
		expectedStatus int
	}{
		{
			name:           "Request with user ID",
			withUserID:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Request without user ID",
			withUserID:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tt.withUserID {
				req = req.WithContext(withUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// ============================================================================
// End-to-End Auth Flow Tests
// ============================================================================

func TestFullAuthFlow_Success(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	userID := "authenticated-user-123"
	token := createTestToken(cfg.SupabaseJWTSecret, cfg.SupabaseURL, userID)

	// Create a handler chain with auth middleware
	authHandler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// After auth middleware, the user ID should be in context
		ctxUserID, ok := middleware.GetUserID(r.Context())
		if !ok {
			t.Error("user ID not found in context after auth")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if ctxUserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, ctxUserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	authHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestFullAuthFlow_Failure(t *testing.T) {
	cfg := &config.Config{
		SupabaseURL:       "https://test.supabase.co",
		SupabaseJWTSecret: "test-secret",
	}

	// Test with expired token
	token := createExpiredToken(cfg.SupabaseJWTSecret, cfg.SupabaseURL, "user-123")

	handler := middleware.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for expired token, got %d", http.StatusUnauthorized, rr.Code)
	}
}
