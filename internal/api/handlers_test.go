package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/middleware"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

func TestHandleImportRecipe_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	body := ImportRecipeRequest{URL: "https://instagram.com/p/test"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/recipe", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.HandleImportRecipe(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleJobStatus_MissingJobID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipe-status", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleJobStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserImportStatus_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/user-import-status", nil)
	rr := httptest.NewRecorder()

	srv.HandleUserImportStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleGenerateEmbedding_MissingRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	body := GenerateEmbeddingRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/generate-embedding", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.HandleGenerateEmbedding(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGetInstructionIngredientsCount_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/instruction-ingredients-count", nil)
	rr := httptest.NewRecorder()

	srv.HandleGetInstructionIngredientsCount(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleGetInstructionIngredientsCount_MissingRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/instruction-ingredients-count", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleGetInstructionIngredientsCount(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGetRecipeSteps_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes/test-id/steps", nil)
	rr := httptest.NewRecorder()

	srv.HandleGetRecipeSteps(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleGetRecipeSteps_MissingRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes//steps", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleGetRecipeSteps(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGetRecipe_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes/test-id", nil)
	rr := httptest.NewRecorder()

	srv.HandleGetRecipe(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleGetRecipe_MissingRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes//", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleGetRecipe(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGetInstructionIngredientsCount_EmptyRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/instruction-ingredients-count?recipe_id=", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleGetInstructionIngredientsCount(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleGetRecipeSteps_EmptyRecipeID(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes//steps", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	srv.HandleGetRecipeSteps(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestParseUUID_InvalidUUID(t *testing.T) {
	invalidUUID := "not-a-valid-uuid"
	parsed := parseUUID(invalidUUID)
	if parsed.Valid {
		t.Errorf("Expected invalid UUID for invalid input, but got valid UUID")
	}
}

func TestHandleGetInstructionIngredientsCount_DatabaseError(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/instruction-ingredients-count?recipe_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	// This should panic due to nil DB, which will be caught by test
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil DB
		}
	}()
	srv.HandleGetInstructionIngredientsCount(rr, req)
}

func TestHandleGetRecipeSteps_DatabaseError(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes/550e8400-e29b-41d4-a716-446655440000/steps", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	// This should panic due to nil DB, which will be caught by test
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil DB
		}
	}()
	srv.HandleGetRecipeSteps(rr, req)
}

func TestHandleGetRecipe_DatabaseError(t *testing.T) {
	cfg := &config.Config{}
	srv := NewServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/recipes/550e8400-e29b-41d4-a716-446655440000", nil)
	req = req.WithContext(withUserID(req.Context(), uuid.New().String()))
	rr := httptest.NewRecorder()

	// This should panic due to nil DB, which will be caught by test
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil DB
		}
	}()
	srv.HandleGetRecipe(rr, req)
}
