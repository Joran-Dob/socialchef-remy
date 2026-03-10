package search

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type CrossEncoderReranker struct {
	openai OpenAIClient
}

func NewCrossEncoderReranker(openai OpenAIClient) *CrossEncoderReranker {
	return &CrossEncoderReranker{openai: openai}
}

func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, results []SearchResult, topK int) ([]SearchResult, error) {
	if len(results) <= topK {
		return results, nil
	}

	candidates := results
	if len(candidates) > 20 {
		candidates = candidates[:20]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Rate how relevant each recipe is to the query: \"%s\"\n\n", query))
	sb.WriteString("Rate 1-10 (10 = perfect match). Only respond with numbers.\n\n")

	for i, r := range candidates {
		sb.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, r.RecipeName, r.Description))
	}

	sb.WriteString("\nScores (one per line):\n")

	response, err := r.callOpenAI(ctx, sb.String())
	if err != nil {
		return results, err
	}

	scores := parseScores(response, len(candidates))

	for i := range candidates {
		if i < len(scores) {
			llmScore := float64(scores[i]) / 10.0
			candidates[i].Similarity = 0.6*candidates[i].Similarity + 0.4*llmScore
		}
	}

	sortResultsBySimilarity(candidates)

	if len(candidates) > topK {
		return candidates[:topK], nil
	}
	return candidates, nil
}

func (r *CrossEncoderReranker) callOpenAI(ctx context.Context, prompt string) (string, error) {
	_ = r.openai
	_ = ctx
	_ = prompt
	return "", nil
}

func parseScores(response string, expectedCount int) []int {
	scores := make([]int, 0, expectedCount)
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if score, err := strconv.Atoi(line); err == nil {
			if score >= 1 && score <= 10 {
				scores = append(scores, score)
			}
		}
	}

	return scores
}

func sortResultsBySimilarity(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
}
