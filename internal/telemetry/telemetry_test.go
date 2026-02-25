package telemetry

import (
	"context"
	"testing"
)

func TestInitTelemetry(t *testing.T) {
	// Test with empty endpoint (should not fail, just no telemetry)
	shutdown, err := InitTelemetry(context.Background(), "test-service", "v1.0.0", "test", "", nil)
	if err != nil {
		t.Fatalf("InitTelemetry failed: %v", err)
	}
	if shutdown != nil {
		defer shutdown(context.Background())
	}
}

func TestTracer(t *testing.T) {
	tracer := Tracer("test-tracer")
	if tracer == nil {
		t.Fatal("Tracer returned nil")
	}
}

func TestMiddleware(t *testing.T) {
	mw := Middleware()
	if mw == nil {
		t.Fatal("Middleware returned nil")
	}
}
