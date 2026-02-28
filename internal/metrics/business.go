package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("socialchef/business")

	// Recipe metrics
	RecipeImportsTotal   metric.Int64Counter
	RecipeImportDuration metric.Float64Histogram

	// External API metrics
	ExternalAPICallsTotal metric.Int64Counter
	ExternalAPIDuration   metric.Float64Histogram

	// AI metrics
	AIGenerationDuration metric.Float64Histogram

	// Provider fallback metrics
	ProviderFallbackTotal metric.Int64Counter
)

func Init() error {
	var err error

	// Recipe metrics
	RecipeImportsTotal, err = meter.Int64Counter(
		"recipe.imports.total",
		metric.WithDescription("Total number of recipe imports"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	RecipeImportDuration, err = meter.Float64Histogram(
		"recipe.import.duration",
		metric.WithDescription("Duration of recipe import process"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2, 5, 10, 30, 60),
	)
	if err != nil {
		return err
	}

	// External API metrics
	ExternalAPICallsTotal, err = meter.Int64Counter(
		"external.api.calls.total",
		metric.WithDescription("Total number of external API calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	ExternalAPIDuration, err = meter.Float64Histogram(
		"external.api.duration",
		metric.WithDescription("Duration of external API calls"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2, 5, 10, 30),
	)
	if err != nil {
		return err
	}

	// AI metrics
	AIGenerationDuration, err = meter.Float64Histogram(
		"ai.generation.duration",
		metric.WithDescription("Duration of AI recipe generation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2, 5, 10, 30, 60),
	)
	if err != nil {
		return err
	}

	// Provider fallback metrics
	ProviderFallbackTotal, err = meter.Int64Counter(
		"provider.fallback.total",
		metric.WithDescription("Total number of provider fallback events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	return nil
}
