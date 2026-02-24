# Fly.io Deployment Guide

This document provides complete instructions for deploying the SocialChef Remy application to Fly.io.

## Overview

**App Name**: `socialchef-remy`  
**Primary Region**: `sjc` (San Jose)  
**Health Check**: `GET /health` on port 8080  

The application runs two processes:
- **Server**: HTTP API server (entrypoint in Dockerfile)
- **Worker**: Asynq background job processor (requires separate process configuration)

---

## Prerequisites

1. [Fly.io CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
2. Logged in: `fly auth login`
3. Access to all required secrets

---

## Required Environment Variables

### Core Required Secrets

These secrets are mandatory for the application to start (enforced by `config.go` validation):

| Variable | Description | Source |
|----------|-------------|--------|
| `DATABASE_URL` | Supabase Postgres connection string | Supabase Dashboard → Settings → Database |
| `SUPABASE_URL` | Supabase project URL | Supabase Dashboard → Settings → API |
| `SUPABASE_JWT_SECRET` | JWT verification secret | Supabase Dashboard → Settings → API → JWT Settings |
| `REDIS_URL` | Redis connection string for Asynq | Fly Redis, Upstash, or self-hosted |

### Optional Secrets

These secrets enable additional features but the app will run without them:

| Variable | Description | Source |
|----------|-------------|--------|
| `SUPABASE_SERVICE_ROLE_KEY` | Service role key for Storage/Realtime API | Supabase Dashboard → Settings → API → service_role key |
| `OPENAI_API_KEY` | OpenAI API access | [OpenAI Dashboard](https://platform.openai.com/api-keys) |
| `GROQ_API_KEY` | Groq API access | [Groq Console](https://console.groq.com/keys) |
| `APIFY_API_KEY` | Apify API for TikTok scraping | [Apify Console](https://console.apify.com/) |
| `PROXY_SERVER_URL` | Instagram proxy server URL | Your proxy provider |
| `PROXY_API_KEY` | Instagram proxy API key | Your proxy provider |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | Your observability provider |

### Built-in Configuration

| Variable | Value | Source |
|----------|-------|--------|
| `PORT` | `8080` | Hardcoded in `fly.toml` |

---

## Setting Up Secrets

### Step 1: Set Required Secrets

**⚠️ IMPORTANT**: Replace placeholder values with actual secrets. Never commit real secrets.

```bash
# Required secrets - MUST be set before first deploy
fly secrets set DATABASE_URL="postgresql://postgres:PASSWORD@db.PROJECT.supabase.co:5432/postgres" --app socialchef-remy
fly secrets set SUPABASE_URL="https://PROJECT.supabase.co" --app socialchef-remy
fly secrets set SUPABASE_JWT_SECRET="YOUR_JWT_SECRET" --app socialchef-remy
fly secrets set REDIS_URL="redis://default:PASSWORD@FLY_REDIS_URL:6379" --app socialchef-remy
```

### Step 2: Set Optional Secrets (as needed)

```bash
# Supabase service role key (needed for Storage/Realtime)
fly secrets set SUPABASE_SERVICE_ROLE_KEY="YOUR_SERVICE_ROLE_KEY" --app socialchef-remy

# AI/LLM providers
fly secrets set OPENAI_API_KEY="sk-..." --app socialchef-remy
fly secrets set GROQ_API_KEY="gsk_..." --app socialchef-remy

# External services
fly secrets set APIFY_API_KEY="apify_api_..." --app socialchef-remy
fly secrets set PROXY_SERVER_URL="https://your-proxy.com" --app socialchef-remy
fly secrets set PROXY_API_KEY="your-proxy-api-key" --app socialchef-remy

# Observability (optional)
fly secrets set OTEL_EXPORTER_OTLP_ENDPOINT="https://your-otel-collector.com" --app socialchef-remy
```

---

## Redis Setup

The application requires Redis for Asynq background job processing.

### Option A: Fly.io Redis (Recommended)

```bash
# Create a Fly Redis instance
fly redis create --name socialchef-remy-redis --region sjc --eviction --no-replicas

# Get the connection string
fly redis status --name socialchef-remy-redis

# Set the REDIS_URL secret (format: redis://default:PASSWORD@HOST:PORT)
fly secrets set REDIS_URL="redis://default:PASSWORD@HOST:6379" --app socialchef-remy
```

### Option B: Upstash Redis

1. Create a Redis database at [Upstash Console](https://console.upstash.com/)
2. Choose region close to `sjc` for lowest latency
3. Copy the Redis connection string
4. Set as `REDIS_URL` secret

### Option C: Self-hosted on Fly

Not recommended for production but viable for development.

---

## Deployment Steps

### 1. Verify Secrets are Set

```bash
# List all secrets (values are hidden)
fly secrets list --app socialchef-remy

# Expected output shows variable names, not values
NAME                          DIGEST                           CREATED AT
DATABASE_URL                  29f7253...                       2024-01-15T10:30:00Z
SUPABASE_URL                  a1b2c3d...                       2024-01-15T10:30:01Z
SUPABASE_JWT_SECRET           e4f5g6h...                       2024-01-15T10:30:02Z
REDIS_URL                     i7j8k9l...                       2024-01-15T10:30:03Z
```

### 2. Deploy the Application

```bash
# Deploy to Fly.io (reads fly.toml from current directory)
fly deploy --app socialchef-remy

# Watch the deployment logs
fly deploy --app socialchef-remy --detach=false
```

### 3. Verify Deployment Status

```bash
# Check app status
fly status --app socialchef-remy

# Expected output shows running machines
Machines
PROCESS ID              VERSION REGION  STATE   ROLE    CHECKS                  LAST UPDATED
app     123456789abcde  1       sjc     started         1 total, 1 passing      2024-01-15T10:35:00Z
```

### 4. Verify Health Check

```bash
# Check health endpoint
curl https://socialchef-remy.fly.dev/health

# Expected response:
# {"status":"ok"} (or similar healthy response)
```

### 5. Monitor Logs

```bash
# View live logs
fly logs --app socialchef-remy

# View logs with filtering
fly logs --app socialchef-remy --json | jq '.message'
```

---

## Worker Process Setup

The Dockerfile builds both `server` and `worker` binaries, but `fly.toml` currently only runs the server. To run the worker:

### Option 1: Separate Fly App for Worker

Create a separate `fly.worker.toml`:

```toml
app = "socialchef-remy-worker"
primary_region = "sjc"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"

[[vm]]
  memory = "512mb"
  cpu_kind = "shared"
  cpus = 1

[processes]
  worker = "/app/worker"
```

Deploy:
```bash
fly deploy --config fly.worker.toml --app socialchef-remy-worker
```

### Option 2: Multiple Processes in Single App (Fly Machines)

Modify `fly.toml` to define multiple processes:

```toml
[processes]
  server = "/app/server"
  worker = "/app/worker"

[[http_service]]
  processes = ["server"]
  internal_port = 8080
  # ... rest of config
```

---

## Verification Checklist

### Before First Deploy

- [ ] `fly.toml` exists and `app` name is correct
- [ ] `Dockerfile` builds successfully locally: `docker build -t test .`
- [ ] All required secrets are set via `fly secrets set`
- [ ] `DATABASE_URL` points to correct Supabase project
- [ ] `REDIS_URL` is accessible from Fly region (`sjc`)

### After Deploy

- [ ] `fly status` shows machine in `started` state
- [ ] Health check passes (1 total, 1 passing)
- [ ] `fly logs` shows successful startup without config errors
- [ ] Application responds to HTTP requests
- [ ] Background jobs are being processed (if worker is running)

---

## Secrets Security

### How Secrets are Protected

1. **Encryption at Rest**: Fly encrypts secrets using AWS KMS
2. **Environment Injection**: Secrets are injected at runtime, not in the image
3. **No Logging**: Secrets are redacted from `fly logs` output
4. **Access Control**: Only app members can read secret metadata (not values)

### Best Practices

1. **Never commit secrets**: Add `.env` files to `.gitignore`
2. **Rotate regularly**: Update secrets periodically via `fly secrets set`
3. **Use least privilege**: Service role keys should have minimal permissions
4. **Separate environments**: Use different Supabase projects for prod/staging

### Checking Logs for Secret Leaks

```bash
# Search logs for potential secret exposure
fly logs --app socialchef-remy | grep -i "password\|secret\|key\|token"

# If you see secrets in logs, check your application code:
# - Never log config values directly
# - Use structured logging that excludes sensitive fields
```

---

## Troubleshooting

### App Won't Start

```bash
# Check recent logs for config errors
fly logs --app socialchef-remy --recent

# Look for:
# - "DATABASE_URL is required"
# - "SUPABASE_URL is required"
# - "SUPABASE_JWT_SECRET is required"
# - "REDIS_URL is required"
```

### Health Check Failing

```bash
# Check health endpoint manually
curl -v https://socialchef-remy.fly.dev/health

# Check if app is listening on correct port
fly ssh console --app socialchef-remy
# Inside container:
netstat -tlnp | grep 8080
```

### Secrets Not Set

```bash
# Verify secrets exist
fly secrets list --app socialchef-remy

# Compare with required list from config.go
# Missing any? Set them:
fly secrets set MISSING_VAR="value" --app socialchef-remy

# Restart app to pick up new secrets
fly deploy --app socialchef-remy
```

### Database Connection Issues

```bash
# Test database connectivity from Fly
fly ssh console --app socialchef-remy
# Inside container:
apk add --no-cache postgresql-client
psql "$DATABASE_URL" -c "SELECT 1;"
```

---

## Useful Commands Reference

```bash
# Secrets management
fly secrets list --app socialchef-remy          # List all secrets (values hidden)
fly secrets set KEY=value --app socialchef-remy # Set a secret
fly secrets unset KEY --app socialchef-remy     # Remove a secret

# Deployment
fly deploy --app socialchef-remy                # Deploy application
fly deploy --app socialchef-remy --build-only   # Build without deploying
fly status --app socialchef-remy                # Check app status

# Monitoring
fly logs --app socialchef-remy                  # View live logs
fly logs --app socialchef-remy --recent         # View recent logs only
fly ssh console --app socialchef-remy           # SSH into running machine

# Scaling (if needed)
fly scale count 2 --app socialchef-remy         # Run 2 machines
fly scale memory 1024 --app socialchef-remy     # Increase memory

# Database (if using Fly Postgres, not Supabase)
fly postgres connect --app socialchef-remy-db   # Connect to Fly Postgres
```

---

## Configuration Files Summary

| File | Purpose |
|------|---------|
| `fly.toml` | Fly.io app configuration (region, services, health checks) |
| `Dockerfile` | Multi-stage build for Go server and worker binaries |
| `internal/config/config.go` | Go config struct and environment variable loading |

---

## Environment Variable Cross-Reference

From `internal/config/config.go`:

| Config Field | Env Var | Required | Default |
|--------------|---------|----------|---------|
| `DatabaseURL` | `DATABASE_URL` | ✅ Yes | - |
| `SupabaseURL` | `SUPABASE_URL` | ✅ Yes | - |
| `SupabaseJWTSecret` | `SUPABASE_JWT_SECRET` | ✅ Yes | - |
| `SupabaseServiceRoleKey` | `SUPABASE_SERVICE_ROLE_KEY` | ❌ No | - |
| `RedisURL` | `REDIS_URL` | ✅ Yes | - |
| `OpenAIKey` | `OPENAI_API_KEY` | ❌ No | - |
| `GroqKey` | `GROQ_API_KEY` | ❌ No | - |
| `ApifyAPIKey` | `APIFY_API_KEY` | ❌ No | - |
| `ProxyServerURL` | `PROXY_SERVER_URL` | ❌ No | - |
| `ProxyAPIKey` | `PROXY_API_KEY` | ❌ No | - |
| `OtelExporterOTLPEndpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | ❌ No | - |
| `Port` | `PORT` | ❌ No | `8080` |

---

## Post-Deployment

After successful deployment:

1. **Set up monitoring**: Configure alerts for health check failures
2. **Enable backups**: Ensure Supabase database backups are configured
3. **Set up CI/CD**: Automate deployments on push to main branch
4. **Document custom domains**: If using custom domain, configure in Fly dashboard
5. **Scale as needed**: Monitor resource usage and adjust VM size in `fly.toml`
