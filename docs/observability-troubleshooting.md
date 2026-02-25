# Observability Troubleshooting

Common issues and solutions for the observability setup.

## Traces Not Appearing

### Symptoms
- No traces in Grafana
- Tempo search returns empty
- Application logs show no trace output

### Diagnosis

1. **Check OTLP endpoint configuration**
   ```bash
   # In docker-compose
   docker-compose exec app env | grep OTEL
   
   # Should show:
   # OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318
   ```

2. **Verify Tempo is running**
   ```bash
   docker-compose ps tempo
   curl http://localhost:3200/ready
   ```

3. **Check app logs for OTel errors**
   ```bash
   docker-compose logs app | grep -i "otel\|telemetry"
   ```

### Solutions

- **Missing endpoint**: Add `OTEL_EXPORTER_OTLP_ENDPOINT` to environment
- **Wrong endpoint**: Ensure it points to Tempo (Docker: `http://tempo:4318`, Local: `http://localhost:4318`)
- **Network issue**: Ensure app and Tempo are on same Docker network

## Missing Span Attributes

### Symptoms
- Traces appear but lack expected attributes
- No HTTP route patterns, just paths
- No database query names

### Diagnosis

1. **Check middleware order**
   ```go
   // Middleware must be applied before handlers
   r.Use(otelchi.Middleware(...))
   r.Get("/api/endpoint", handler) // After middleware
   ```

2. **Verify instrumentation is imported**
   ```bash
   grep -r "otelchi\|otelpgx\|otelhttp" internal/
   ```

### Solutions

- **HTTP routes**: Ensure `otelchi.WithChiRoutes(r)` is passed
- **Database**: Verify `otelpgx.NewTracer()` is set on connection config
- **HTTP clients**: Use `httpclient.InstrumentedClient` instead of `http.DefaultClient`

## Grafana Not Loading

### Symptoms
- Cannot access http://localhost:3000
- 502 Bad Gateway
- Connection refused

### Diagnosis

1. **Check container status**
   ```bash
   docker-compose ps grafana
   ```

2. **Check port availability**
   ```bash
   lsof -i :3000
   ```

3. **Check Grafana logs**
   ```bash
   docker-compose logs grafana
   ```

### Solutions

- **Port in use**: Stop conflicting service or change port in docker-compose.yml
- **Container not running**: `docker-compose up -d grafana`
- **Permission issue**: Check `./data/grafana` directory permissions

## Tempo Not Receiving Traces

### Symptoms
- App sends traces but Tempo shows none
- OTLP endpoint returns errors

### Diagnosis

1. **Test OTLP endpoint**
   ```bash
   curl http://localhost:4318/v1/traces
   ```

2. **Check Tempo configuration**
   ```bash
   cat tempo.yaml
   ```

3. **Check Tempo logs**
   ```bash
   docker-compose logs tempo
   ```

### Solutions

- **Wrong config**: Verify `tempo.yaml` has OTLP HTTP receiver on `0.0.0.0:4318`
- **Connection refused**: Ensure Tempo container is running
- **Firewall**: Check Docker network configuration

## Logs Missing Trace IDs

### Symptoms
- Logs appear in Loki but without trace_id
- Cannot correlate logs to traces

### Diagnosis

1. **Check logger configuration**
   ```go
   // internal/logger/logger.go should have WithTraceContext
   ```

2. **Verify OTel context propagation**
   ```bash
   # Make request and check logs
   curl http://localhost:8080/health
   docker-compose logs app | grep trace
   ```

### Solutions

- **Logger not using OTel**: Ensure `WithTraceContext(ctx)` is called
- **Context not propagated**: Pass `ctx` through all function calls

## High Memory Usage

### Symptoms
- Tempo/Grafana containers using excessive memory
- OOM kills

### Solutions

1. **Limit trace retention**
   ```yaml
   # tempo.yaml
   retention: 24h
   ```

2. **Reduce sampling** (for high-traffic)
   - Currently at 100% sampling
   - Consider reducing for production

3. **Limit container memory**
   ```yaml
   # docker-compose.yml
   services:
     tempo:
       deploy:
         resources:
           limits:
             memory: 512M
   ```

## Worker Traces Not Linked to HTTP

### Symptoms
- HTTP request trace and worker trace are separate
- Cannot see full request flow

### Explanation

This is expected behavior. Asynq jobs are processed asynchronously, so the worker creates a new trace context. The traces can be linked via:
- Task metadata containing trace information
- Custom attributes added during enqueue

### Solution (Future Enhancement)

To link traces, add trace context to task metadata:
```go
// When enqueuing
task := asynq.NewTask(TypeProcessRecipe, payload)
span := trace.SpanFromContext(ctx)
if span.SpanContext().IsValid() {
    // Add trace context to task metadata
}
```

## Common Error Messages

### "failed to instrument Redis client"
- **Cause**: redisotel initialization failed
- **Solution**: Check Redis connection and credentials

### "context deadline exceeded"
- **Cause**: OTLP export timeout
- **Solution**: Increase timeout or check network connectivity

### "no active span"
- **Cause**: OTel context not propagated
- **Solution**: Ensure `ctx` is passed through call chain
