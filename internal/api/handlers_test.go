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
