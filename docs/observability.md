# Observability Architecture

This document describes the observability architecture for SocialChef Remy.

## Overview

SocialChef Remy uses OpenTelemetry (OTel) for distributed tracing, structured logging, and metrics collection. The architecture follows the "three pillars of observability" pattern with unified telemetry.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        SocialChef Remy                          │
│                                                                 │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐    │
│  │ HTTP     │   │ Worker   │   │ Database │   │ External │    │
│  │ Server   │   │ (Asynq)  │   │ (pgx)    │   │ APIs     │    │
│  │          │   │          │   │          │   │          │    │
│  │ otelchi  │   │ OTel     │   │ otelpgx  │   │ otelhttp │    │
│  │          │   │ Middleware│   │          │   │          │    │
│  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘    │
│       │              │              │              │           │
│       └──────────────┴──────────────┴──────────────┘           │
│                              │                                  │
│                    OpenTelemetry SDK                            │
│                              │                                  │
└──────────────────────────────┼──────────────────────────────────┘
                               │
                    OTLP (HTTP/4318)
                               │
        ┌──────────────────────┴──────────────────────┐
        │                                             │
        ▼                                             ▼
┌───────────────┐                           ┌─────────────────┐
│  Local Dev    │                           │  Production     │
│               │                           │                 │
│  ┌─────────┐  │                           │  ┌───────────┐  │
│  │  Tempo  │  │                           │  │  Grafana  │  │
│  │ (local) │  │                           │  │  Cloud    │  │
│  └─────────┘  │                           │  └───────────┘  │
│  ┌─────────┐  │                           │                 │
│  │  Loki   │  │                           │                 │
│  │ (local) │  │                           │                 │
│  └─────────┘  │                           │                 │
│  ┌─────────┐  │                           │                 │
│  │Prometheus│ │                           │                 │
│  │ (local) │  │                           │                 │
│  └─────────┘  │                           │                 │
│  ┌─────────┐  │                           │                 │
│  │ Grafana │  │                           │                 │
│  │ (local) │  │                           │                 │
│  └─────────┘  │                           │                 │
└───────────────┘                           └─────────────────┘
```

## Instrumented Components

### HTTP Server (Chi Router)
- **Library**: `github.com/riandyrn/otelchi`
- **Middleware**: `otelchi.Middleware`
- **Attributes**: HTTP method, route pattern, status code
- **Health check filtering**: `/health` endpoint excluded from tracing

### Background Worker (Asynq)
- **Library**: Custom middleware using OpenTelemetry SDK
- **File**: `internal/worker/tracing.go`
- **Attributes**: Job type, queue name, retry count, job ID

### Database (PostgreSQL via pgx)
- **Library**: `github.com/exaring/otelpgx`
- **Configuration**: `config.ConnConfig.Tracer = otelpgx.NewTracer()`
- **Attributes**: Query (sanitized), duration, error status

### External HTTP Clients
- **Library**: `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- **File**: `internal/httpclient/client.go`
- **Attributes**: Provider name, URL path, method, status code
- **Providers**: OpenAI, Groq, Apify, Instagram Proxy

### Redis (go-redis)
- **Library**: `github.com/redis/go-redis/extra/redisotel/v9`
- **Configuration**: `redisotel.InstrumentTracing(rdb)`
- **Attributes**: Command name, duration

## Trace Flow

### Recipe Import Flow
```
1. HTTP POST /api/recipe
   └── otelchi creates root span
   
2. Handler validates request
   └── No child span (application logic)
   
3. Handler enqueues job to Asynq
   └── Redis SET span (redisotel)
   
4. Worker picks up job
   └── OTelMiddleware creates job span
   └── May be linked to HTTP trace via trace context
   
5. Worker scrapes social media
   └── HTTP client span (otelhttp) - Apify/Proxy
   
6. Worker transcribes video
   └── HTTP client span (otelhttp) - OpenAI
   
7. Worker generates recipe
   └── HTTP client span (otelhttp) - Groq
   
8. Worker saves to database
   └── Database query spans (otelpgx)
```

## Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint URL | `http://localhost:4318` |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth headers | `Authorization=Bearer token` |
| `ENV` | Environment name | `development` / `production` |
| `SERVICE_NAME` | Service identifier | `socialchef-remy` |
| `SERVICE_VERSION` | Version string | `1.0.0` |

### Local Development
- Endpoint: `http://tempo:4318` (Docker) or `http://localhost:4318`
- No authentication required

### Production (Grafana Cloud)
- Endpoint: From Grafana Cloud settings
- Authentication: API key in headers

## Dashboards

Pre-built dashboards are available in `dashboards/`:
- **HTTP Overview**: Request rate, latency, error rate
- **Worker Overview**: Job processing rate, queue depth
- **Database Overview**: Query duration, connection pool

Import via Grafana → Dashboards → Import.

## Related Documentation

- [Local Development Observability](./observability-local.md)
- [Grafana Cloud Setup](./observability-grafana-cloud.md)
- [Troubleshooting](./observability-troubleshooting.md)
- [Validation Guide](./observability-validation.md)
