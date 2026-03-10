package search

import (
	"context"
	"fmt"
)

// QueryExpander expands search queries with synonyms and related terms
type QueryExpander struct {
	openai OpenAIClient
}

// NewQueryExpander creates a new query expander
func NewQueryExpander(openai OpenAIClient) *QueryExpander {
	return &QueryExpander{openai: openai}
}

// ExpandQuery uses OpenAI to expand the query with synonyms
func (e *QueryExpander) ExpandQuery(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`Expand this recipe search query with synonyms and related terms.
Original query: "%s"

Provide 3-5 relevant search terms or phrases that would help find similar recipes.
Include translations if the query is in Dutch.
Return only the expanded terms separated by commas, no explanations.

Example:
Query: "krentenbollen"
Result: "krentenbollen, Dutch currant buns, rozijnen broodjes, currant bread rolls, Dutch sweet buns"`, query)

	expanded, err := e.openai.Complete(ctx, prompt)
	if err != nil {
		return query, err // Return original on error
	}

	// Combine original with expanded
	return query + " " + expanded, nil
}
