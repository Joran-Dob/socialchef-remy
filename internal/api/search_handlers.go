package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// SearchRequest represents a request to search recipes
type SearchRequest struct {
	Query         string  `json:"query"`
	Limit         int32   `json:"limit,omitempty"`
	MinSimilarity float64 `json:"min_similarity,omitempty"`
}

// HandleSearch performs hybrid search (semantic + text)
// Combines vector similarity (70%) with full-text search (30%)
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
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

	results, err := s.search.SearchHybrid(r.Context(), req.Query, limit)
	if err != nil {
		slog.Error("SearchHybrid failed", "error", err, "query", req.Query)
		http.Error(w, "Failed to perform search: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter results below minimum similarity threshold
	if req.MinSimilarity > 0 {
		idx := 0
		for _, r := range results {
			if r.Similarity >= req.MinSimilarity {
				results[idx] = r
				idx++
			}
		}
		results = results[:idx]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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
