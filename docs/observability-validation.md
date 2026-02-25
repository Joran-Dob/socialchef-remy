# Observability Validation

This document describes how to validate that the observability setup is working correctly.

## Prerequisites

- Docker and Docker Compose installed
- All services configured with correct environment variables

## Starting the Stack

```bash
# Start all services including observability stack
docker-compose up -d

# Verify all services are running
docker-compose ps
```

Expected services:
- `app` - Main application
- `redis` - Redis for task queue
- `grafana` - Visualization UI (port 3000)
- `tempo` - Trace storage (OTLP port 4318)
- `loki` - Log aggregation (port 3100)
- `prometheus` - Metrics collection (port 9090)

## Generating Test Traffic

```bash
# Health check (generates HTTP trace)
curl http://localhost:8080/health

# API request (requires auth token)
curl -X POST http://localhost:8080/api/recipe \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.tiktok.com/@user/video/123"}'
```

## Checking Traces

### Option 1: Tempo API

```bash
# Search for traces
curl http://localhost:3200/api/search

# Get trace by ID
curl http://localhost:3200/api/traces/<trace_id>
```

### Option 2: Grafana UI

1. Open Grafana at http://localhost:3000
2. Navigate to **Explore**
3. Select **Tempo** datasource
4. Use TraceQL to search:
   ```
   { service.name = "socialchef-server" }
   ```

## Expected Trace Structure

### HTTP Request Trace
```
HTTP /api/recipe (socialchef-server)
├── db.query: SELECT (socialchef-remy)
├── http.request: OpenAI (httpclient)
├── http.request: Groq (httpclient)
└── redis.command (redisotel)
```

### Background Job Trace
```
job:process_recipe (socialchef-remy-worker)
├── db.query: INSERT (socialchef-remy)
├── db.query: SELECT (socialchef-remy)
└── http.request: Supabase (httpclient)
```

## Verifying Log Correlation

Logs should contain trace IDs for correlation:

```bash
# Check app logs for trace_id
docker-compose logs app | grep -i trace_id

# Example output:
# {"level":"INFO","msg":"Processing request","trace":{"trace_id":"abc123","span_id":"def456"}}
```

## Verifying All Instrumentation

### HTTP Server (otelchi)
- [ ] HTTP requests generate spans
- [ ] Route patterns are captured (not just paths)
- [ ] HTTP status codes are recorded
- [ ] Errors create error spans

### Worker (OTelMiddleware)
- [ ] Background jobs generate spans
- [ ] Job type is recorded
- [ ] Queue name is recorded
- [ ] Retry count is recorded

### Database (otelpgx)
- [ ] SQL queries generate spans
- [ ] Query duration is captured
- [ ] Errors are marked

### External APIs (otelhttp)
- [ ] HTTP calls generate spans
- [ ] Provider name is recorded
- [ ] Status codes are captured

### Redis (redisotel)
- [ ] Redis commands generate spans
- [ ] Command names are recorded

## Troubleshooting

### No Traces Appearing

1. **Check OTLP endpoint**:
   ```bash
   # Should be set in docker-compose or .env
   echo $OTEL_EXPORTER_OTLP_ENDPOINT
   ```

2. **Verify Tempo is receiving**:
   ```bash
   curl http://localhost:3200/ready
   ```

3. **Check app logs for OTel errors**:
   ```bash
   docker-compose logs app | grep -i otel
   ```

### Missing Span Attributes

1. **Verify middleware order** - OTel middleware should be early in the chain
2. **Check instrumentation is applied** - Look for imports in main.go

### Grafana Not Loading

1. **Check container status**:
   ```bash
   docker-compose ps grafana
   ```

2. **Verify port is available**:
   ```bash
   lsof -i :3000
   ```

3. **Check Grafana logs**:
   ```bash
   docker-compose logs grafana
   ```

### Tempo Not Receiving Traces

1. **Verify OTLP endpoint**:
   ```bash
   curl http://localhost:4318/v1/traces
   ```

2. **Check Tempo configuration**:
   ```bash
   cat tempo.yaml
   ```

## Validation Checklist

After completing setup, verify:

- [ ] `docker-compose up -d` starts all services
- [ ] Grafana accessible at http://localhost:3000
- [ ] HTTP request to `/health` generates a trace
- [ ] Trace appears in Tempo/Grafana within 30 seconds
- [ ] Logs contain trace IDs
- [ ] All instrumented components show in traces
