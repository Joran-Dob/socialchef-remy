package search

import (
	"strings"
)

// SearchIntent represents the type of search query
type SearchIntent string

const (
	IntentByName       SearchIntent = "by_name"       // "carbonara recipe"
	IntentByIngredient SearchIntent = "by_ingredient" // "recipes with eggs"
	IntentSimilarTo    SearchIntent = "similar_to"    // "something like pasta"
	IntentQuick        SearchIntent = "quick"         // "quick dinner"
	IntentCuisine      SearchIntent = "cuisine"       // "Italian food"
	IntentGeneral      SearchIntent = "general"       // default
)

// QueryClassifier detects search intent
type QueryClassifier struct{}

// NewQueryClassifier creates a new classifier
func NewQueryClassifier() *QueryClassifier {
	return &QueryClassifier{}
}

// Classify determines the search intent from the query
func (c *QueryClassifier) Classify(query string) SearchIntent {
	lowerQuery := strings.ToLower(query)

	// Check for ingredient patterns
	ingredientPatterns := []string{"with", "using", "contain", "have"}
	for _, pattern := range ingredientPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return IntentByIngredient
		}
	}

	// Check for similarity patterns
	similarityPatterns := []string{"like", "similar", "something"}
	for _, pattern := range similarityPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return IntentSimilarTo
		}
	}

	// Check for quick/easy patterns
	quickPatterns := []string{"quick", "fast", "easy", "simple", "15 min", "30 min"}
	for _, pattern := range quickPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return IntentQuick
		}
	}

	// Check for cuisine patterns
	cuisines := []string{"italian", "dutch", "french", "mexican", "asian", "chinese", "japanese", "indian"}
	for _, cuisine := range cuisines {
		if strings.Contains(lowerQuery, cuisine) {
			return IntentCuisine
		}
	}

	// Check for specific recipe name (short query, no spaces)
	if len(strings.Fields(query)) <= 2 {
		return IntentByName
	}

	return IntentGeneral
}
