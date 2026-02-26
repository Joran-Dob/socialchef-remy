package api

import (
	"encoding/json"
	"net/http"
)

// SearchRequest represents a request to search recipes
type SearchRequest struct {
	Query string `json:"query"`
	Limit int32  `json:"limit,omitempty"`
}

// HandleSearch performs hybrid search (semantic + text)
// Currently it uses semantic search as the underlying implementation
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	s.HandleSearchSemantic(w, r)
}

// HandleSearchSemantic performs semantic (vector) search
func (s *Server) HandleSearchSemantic(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10 // Default limit
	} else if limit > 50 {
		limit = 50 // Max limit
	}

	results, err := s.search.SearchSemantic(r.Context(), req.Query, limit)
	if err != nil {
		http.Error(w, "Failed to perform search", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleSearchByName performs text-based search on recipe names
func (s *Server) HandleSearchByName(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10 // Default limit
	} else if limit > 50 {
		limit = 50 // Max limit
	}

	results, err := s.search.SearchByName(r.Context(), req.Query, limit)
	if err != nil {
		http.Error(w, "Failed to perform search", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
