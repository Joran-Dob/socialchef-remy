# Development Setup

This guide gets you running Remy locally for development. For project overview and features, see [README.md](README.md).

## Prerequisites

- **Go 1.25+**: [Install Go](https://golang.org/dl/)
- **Docker**: For running Redis locally ([Install Docker](https://docs.docker.com/get-docker/))
- **Hosted Supabase Account**: Remy uses hosted Supabase for PostgreSQL and Storage. Sign up at [supabase.com](https://supabase.com)
- **API Keys**: You'll need accounts and API keys for:
  - OpenAI (for transcription)
  - Groq and/or Cerebras (for recipe generation)
  - Apify (for TikTok scraping)
  - A proxy service (for Instagram scraping)

## Quick Start

### 1. Clone and Setup Environment

```bash
# Clone the repository
git clone <repo-url>
cd socialchef-remy

# Copy environment template
cp .env.example .env
```

Edit `.env` and fill in your actual API keys and Supabase credentials.

### 2. Start Redis

```bash
# Start Redis using Docker Compose
docker-compose -f docker-compose.dev.yml up -d

# Verify Redis is running
docker-compose -f docker-compose.dev.yml ps
```

### 3. Run the Application

```bash
# Install dependencies
go mod download

# Run the server
go run main.go

# Or with air for hot reload (if installed)
air
```

The server will start on `http://localhost:8080` (or the PORT you configured).

## Environment Variables

| Variable | Required | Description |
| :--- | :--- | :--- |
| `DATABASE_URL` | Yes | PostgreSQL connection string. Format: `postgres://user:password@host:port/database`. For hosted Supabase, get this from your project settings. |
| `SUPABASE_URL` | Yes | Your Supabase project URL (e.g., `https://your-project.supabase.co`). Find this in Project Settings > API. |
| `SUPABASE_SERVICE_ROLE_KEY` | Yes | Admin key for Supabase operations. **Keep this secret.** Get it from Project Settings > API > service_role key. |
| `SUPABASE_JWT_SECRET` | No | JWT secret for token verification. Only needed if implementing auth. |
| `REDIS_URL` | Yes | Redis connection string. For local Docker setup, use `localhost:6379`. |
| `OPENAI_API_KEY` | Yes | OpenAI API key for video transcription and embeddings. Get from [platform.openai.com](https://platform.openai.com). |
| `GROQ_API_KEY` | Yes | Groq API key for recipe generation (fast, cheap). Get from [groq.com](https://groq.com). |
| `CEREBRAS_API_KEY` | No | Cerebras API key for recipe generation (alternative to Groq). Get from [cerebras.ai](https://cerebras.ai). |
| `APIFY_API_KEY` | Yes | Apify API key for TikTok scraping. Get from [apify.com](https://apify.com). |
| `PROXY_SERVER_URL` | Yes | Proxy server URL for Instagram scraping (required to avoid blocks). |
| `PROXY_API_KEY` | Yes | Proxy authentication key. |
| `PORT` | No | Server port. Defaults to `8080`. |
| `ENV` | No | Environment name. Use `development` for local dev. |
| `SERVICE_NAME` | No | Service identifier for observability. |
| `SERVICE_VERSION` | No | Version string for observability. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | OpenTelemetry collector endpoint for traces. |
| `OTEL_EXPORTER_OTLP_HEADERS` | No | Headers for OTLP authentication (e.g., Grafana Cloud). |

## Architecture Overview

Remy uses a hybrid architecture for data persistence:

### Hosted Supabase (Cloud)

PostgreSQL database and file storage run on Supabase's hosted service. This gives you:

- Managed PostgreSQL with automatic backups
- Built-in storage for recipe images and videos
- Connection pooling and SSL out of the box
- Row-level security policies (if needed later)

**Setup**: Create a project at supabase.com, then copy the connection details into your `.env` file.

### Local Redis (Docker)

The task queue (powered by Asynq) runs on Redis in a local Docker container:

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│   Your Go App   │────▶│  Local Redis │────▶│  Background Jobs │
│   (localhost)   │     │  (Docker)    │     │  (Asynq workers) │
└─────────────────┘     └──────────────┘     └──────────────────┘
          │
          ▼
┌──────────────────────────────────────────────────────────────┐
│              Hosted Supabase (PostgreSQL + Storage)          │
└──────────────────────────────────────────────────────────────┘
```

This setup keeps your data persistent in the cloud while keeping the task queue fast and local.

## Troubleshooting

### Redis Connection Refused

**Error**: `dial tcp localhost:6379: connect: connection refused`

**Fix**:
```bash
# Check if Redis container is running
docker-compose -f docker-compose.dev.yml ps

# If not running, start it
docker-compose -f docker-compose.dev.yml up -d

# Check logs if it keeps failing
docker-compose -f docker-compose.dev.yml logs redis
```

### Database Connection Errors

**Error**: `connection refused` or `password authentication failed`

**Fix**:
1. Verify your `DATABASE_URL` in `.env`
2. Check that the Supabase project is active (not paused due to inactivity)
3. Ensure you're using the connection string from Project Settings > Database > Connection string > URI
4. If using IPv4, make sure "Use connection pooling" is enabled in Supabase

### Missing API Keys

**Error**: `API key not set` or `401 Unauthorized`

**Fix**: Double-check all API keys in `.env`. The application validates required keys on startup and will fail fast if any are missing.

### Port Already in Use

**Error**: `bind: address already in use`

**Fix**:
```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process or change PORT in .env
PORT=8081 go run main.go
```

### Module Download Failures

**Error**: `go: module lookup disabled` or proxy errors

**Fix**:
```bash
# Ensure Go module proxy is enabled
go env -w GOPROXY=https://proxy.golang.org,direct

# Clear module cache if corrupted
go clean -modcache
go mod download
```

### Asynq Tasks Not Processing

**Fix**:
1. Check Redis is running: `redis-cli ping` should return `PONG`
2. Verify `REDIS_URL` in `.env` matches your setup
3. Check application logs for worker startup messages
4. Ensure your Asynq server and worker are using the same Redis instance
