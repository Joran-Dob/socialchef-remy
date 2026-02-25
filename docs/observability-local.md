# Local Development Observability

This guide explains how to use the observability stack during local development.

## Quick Start

```bash
# Start all services including observability
docker-compose up -d

# Check all services are running
docker-compose ps
```

## Accessing Services

| Service | URL | Purpose |
|---------|-----|---------|
| Grafana | http://localhost:3000 | Visualization & dashboards |
| Tempo API | http://localhost:3200 | Trace queries |
| Loki API | http://localhost:3100 | Log queries |
| Prometheus | http://localhost:9090 | Metrics queries |

## Grafana Setup

Grafana is pre-configured with datasources:

1. **Open Grafana**: http://localhost:3000
2. **Login**: Anonymous admin access enabled
3. **Datasources**: Pre-configured (Tempo, Loki, Prometheus)

### Viewing Traces

1. Navigate to **Explore**
2. Select **Tempo** datasource
3. Use TraceQL to search:
   ```
   { service.name = "socialchef-server" }
   ```
4. Click on a trace to see the waterfall view

### Viewing Logs

1. Navigate to **Explore**
2. Select **Loki** datasource
3. Use LogQL to search:
   ```
   { job = "app" }
   ```
4. Logs with trace IDs can be linked to traces

## Generating Test Data

### HTTP Requests

```bash
# Simple health check
curl http://localhost:8080/health

# API request (requires valid token)
curl -X POST http://localhost:8080/api/recipe \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.tiktok.com/@user/video/123456"}'
```

### Background Jobs

Jobs are automatically enqueued when you make API requests. To see worker traces:
1. Make an API request that enqueues a job
2. Wait for the worker to process it
3. Search for traces with `job:` prefix

## Configuration

The app is configured to send traces to Tempo via environment variable:

```yaml
# docker-compose.yml
app:
  environment:
    - OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318
```

To disable tracing locally, remove or comment out this variable.

## Debugging

### Check if app is sending traces

```bash
# Look for OTel initialization in logs
docker-compose logs app | grep -i otel
```

### Check Tempo is receiving

```bash
# Tempo ready check
curl http://localhost:3200/ready

# Search for traces
curl http://localhost:3200/api/search
```

### Check Loki is receiving logs

```bash
# Loki ready check
curl http://localhost:3100/ready
```

## Stopping the Stack

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (clears all data)
docker-compose down -v
```

## Data Persistence

Observability data is stored in Docker volumes:
- `./data/grafana` - Grafana dashboards and settings
- `./data/redis` - Redis data (for Asynq)
- Tempo stores traces in `/tmp/tempo/blocks` (ephemeral)

To reset all data:
```bash
docker-compose down -v
rm -rf ./data/grafana
```
