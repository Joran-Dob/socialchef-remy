# Go Rebuild: SocialChef Backend

## TL;DR

> **Quick Summary**: Rebuild the Supabase Edge Functions + Trigger.dev backend as a single Go service deployed to Fly.io. Keep Supabase for Postgres, Auth, Realtime, and Storage. Replace only the compute layer with a performant Go implementation using chi router, Asynq workers, and sqlc for database access.
> 
> **Deliverables**:
> - Go HTTP API server with 4 endpoints (recipe, recipe-status, user-import-status, generate-embedding)
> - Asynq background worker with full recipe processing pipeline
> - Docker Compose for local development
> - Fly.io deployment configuration with Redis
> - OpenTelemetry monitoring integration
> - Unit and integration tests
>
> **Estimated Effort**: Large (15-25 tasks across 4 waves)
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Project scaffold → sqlc queries → Auth middleware → Asynq worker → Recipe endpoint → Deployment

---

## Context

### Original Request
Rebuild `/Users/jorandob/Documents/Projects/SocialChef/socialchef-supabase` as a Go service in `/Users/jorandob/Documents/Projects/SocialChef/socialchef-remy`. Fresh rebuild (not a copy), same functionality, performant, deployed to fly.io, with testing, local dev environment, and monitoring.

### Interview Summary

**Key Discussions**:
- **Database/Infra**: Keep Supabase for Postgres, Auth, Realtime, Storage — Go only replaces compute
- **HTTP Framework**: stdlib + chi (lightweight, idiomatic)
- **Background Jobs**: Asynq with Redis (proven at scale)
- **DB Access**: sqlc (type-safe SQL → Go code generation)
- **Monitoring**: OpenTelemetry + Grafana Cloud + Fly.io built-in metrics
- **Testing**: Tests-after with go test, table-driven tests
- **Realtime Architecture**: Asynq worker pushes progress directly to Supabase Realtime — `recipe-bridge` endpoint ELIMINATED
- **RLS Strategy**: Service role connection + manual `WHERE created_by = $userId` in queries
- **Transactional Saves**: YES — wrap all recipe inserts in a single transaction
- **Dead Code**: EXCLUDE Anthropic and Replicate (never used in current system)
- **Cleanup Cron**: INCLUDE daily cleanup job via Asynq scheduler

**Research Findings** (from explore agent + Metis):
- Current system has 5 Edge Functions, but `recipe-bridge` is unnecessary in Go architecture
- Auth currently only DECODES JWT (no signature verification) — Go must properly verify
- Recipe saves are non-transactional (8+ sequential inserts) — Go will make transactional
- Model names differ from draft: uses `gpt-4o-mini-transcribe`, `gpt-3.5-turbo-1106`, `openai/gpt-oss-120b`, `text-embedding-ada-002`
- Instagram scraping uses proxy + GraphQL API
- TikTok scraping uses Apify actor
- pgvector requires custom sqlc type handling

### Metis Review

**Identified Gaps** (addressed):
- **recipe-bridge architecture**: Worker pushes directly to Realtime, bridge eliminated
- **RLS strategy**: Service role + manual WHERE clauses
- **Transactional saves**: Confirmed transactional
- **Dead code**: Anthropic/Replicate excluded
- **Cleanup cron**: Included via Asynq scheduler
- **Model names**: Use actual model names from code, not draft approximations
- **JWT verification**: Must verify signatures (not just decode)
- **Retry logic**: Internal retries for transient errors, Asynq retries for crashes only

---

## Work Objectives

### Core Objective
Build a production-ready Go backend service that replaces Supabase Edge Functions + Trigger.dev, achieving full feature parity while improving reliability (transactional saves, proper JWT verification).

### Concrete Deliverables
- `cmd/server/main.go` — HTTP server entrypoint
- `cmd/worker/main.go` — Asynq worker entrypoint  
- `internal/api/` — HTTP handlers (chi routes)
- `internal/worker/` — Asynq task handlers
- `internal/db/` — sqlc generated database code
- `internal/services/` — Business logic (scrapers, AI clients, storage)
- `internal/middleware/` — Auth middleware with JWT verification
- `docker-compose.yml` — Local dev environment
- `Dockerfile` — Production container
- `fly.toml` — Fly.io deployment config
- `go.mod`, `sqlc.yaml`, `Makefile`

### Definition of Done
- [ ] All 4 API endpoints return identical response shapes to current system
- [ ] Recipe pipeline processes Instagram and TikTok URLs end-to-end
- [ ] Progress updates broadcast to Supabase Realtime during processing
- [ ] JWT auth middleware properly verifies Supabase tokens
- [ ] Local dev environment starts with `docker-compose up`
- [ ] Deployed to Fly.io with Redis
- [ ] OpenTelemetry traces visible in Grafana Cloud
- [ ] All unit tests pass with `go test ./...`

### Must Have
- Feature parity with current system (excluding CMS)
- Proper JWT signature verification
- Transactional recipe saves
- Realtime progress broadcasting from worker
- Daily cleanup cron job

### Must NOT Have (Guardrails)
- NO `recipe-bridge` endpoint — eliminated in new architecture
- NO Anthropic or Replicate clients — dead code
- NO database migrations — schema stays in Supabase
- NO WebSocket server — use Supabase Realtime REST API
- NO API versioning — single version, no prefix
- NO rate limiting, admin endpoints, or Swagger generation — not in current system
- NO factory/strategy patterns for scrapers — keep it flat and explicit

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: NO (greenfield project)
- **Automated tests**: Tests-after
- **Framework**: go test with table-driven tests
- **Coverage target**: Core business logic (pipeline, scrapers, AI clients)

### QA Policy
Every task includes agent-executed QA scenarios:
- **API endpoints**: curl commands with exact assertions
- **Worker pipeline**: Submit URL, verify database state, verify Realtime broadcasts
- **Auth**: Test valid/expired/missing tokens

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — 7 tasks, all parallel):
├── T1: Project scaffold (go mod, Makefile, .env.example) [quick]
├── T2: Docker Compose local dev setup [quick]
├── T3: sqlc configuration + connection pool [quick]
├── T4: Database queries (all CRUD operations) [quick]
├── T5: Configuration management (env vars, types) [quick]
├── T6: OpenTelemetry setup [quick]
└── T7: Supabase JWT auth middleware [quick]

Wave 2 (Core Services — 6 tasks, parallel after Wave 1):
├── T8: Asynq server + client setup [quick]
├── T9: OpenAI client (transcribe, extract, embed) [unspecified-high]
├── T10: Groq client (validation, alternative extraction) [unspecified-high]
├── T11: Instagram scraper (proxy + GraphQL) [unspecified-high]
├── T12: TikTok scraper (Apify actor) [unspecified-high]
└── T13: Supabase Storage client [quick]

Wave 3 (Business Logic — 6 tasks, parallel after Wave 2):
├── T14: Recipe processing pipeline (Asynq handler) [deep]
├── T15: Realtime progress broadcaster [quick]
├── T16: Cleanup cron job (Asynq scheduler) [quick]
├── T17: POST /recipe endpoint [quick]
├── T18: POST /recipe-status endpoint [quick]
└── T19: GET /user-import-status endpoint [quick]

Wave 4 (Finalization — 5 tasks, parallel after Wave 3):
├── T20: POST /generate-embedding endpoint [quick]
├── T21: Integration tests [deep]
├── T22: Fly.io deployment (Dockerfile, fly.toml, Redis) [quick]
├── T23: Production config + secrets [quick]
└── T24: End-to-end verification [deep]

Critical Path: T1 → T3 → T4 → T8 → T14 → T17 → T22 → T24
```

### Agent Dispatch Summary
- **Wave 1**: 7 tasks → all `quick`
- **Wave 2**: 6 tasks → 2 `quick`, 4 `unspecified-high`
- **Wave 3**: 6 tasks → 4 `quick`, 1 `deep`, 1 depends on Wave 2
- **Wave 4**: 5 tasks → 2 `quick`, 2 `deep`

---

## TODOs

### Wave 1: Foundation (7 tasks — all parallel)

- [ ] 1. Project Scaffold

  **What to do**:
  - Initialize Go module: `go mod init github.com/socialchef/remy`
  - Create directory structure: `cmd/server/`, `cmd/worker/`, `internal/api/`, `internal/worker/`, `internal/db/`, `internal/services/`, `internal/middleware/`, `internal/config/`
  - Create `Makefile` with targets: `build`, `test`, `sqlc-generate`, `docker-up`, `docker-down`
  - Create `.env.example` with all required environment variables
  - Create `.gitignore` for Go

  **Must NOT do**:
  - Do NOT create any business logic files yet
  - Do NOT add unnecessary abstractions

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T2-T7 all depend on this for directory structure
  - **Blocked By**: None

  **References**:
  - Go project layout: `cmd/` for entrypoints, `internal/` for private packages
  - Makefile pattern: `go build -o bin/server ./cmd/server`

  **Acceptance Criteria**:
  - [ ] `go mod tidy` succeeds
  - [ ] All directories exist
  - [ ] `make build` compiles (even if empty mains)
  - [ ] `.env.example` lists all required vars

  **QA Scenarios**:
  ```
  Scenario: Project structure is valid
    Tool: Bash
    Steps:
      1. cd /Users/jorandob/Documents/Projects/SocialChef/socialchef-remy
      2. go mod tidy
      3. make build
    Expected Result: Both commands exit with code 0
    Evidence: .sisyphus/evidence/task-01-structure.txt
  ```

  **Commit**: YES
  - Message: `feat(scaffold): initial project structure`
  - Files: `go.mod`, `Makefile`, `.gitignore`, `.env.example`

- [ ] 2. Docker Compose Local Dev

  **What to do**:
  - Create `docker-compose.yml` with services: `app` (Go), `redis` (for Asynq)
  - Create `Dockerfile.dev` for local development with hot reload (use `air` or simple volume mount)
  - Configure Redis to persist data locally
  - Add `make docker-up` and `make docker-down` targets
  - Service `app` should connect to Supabase via environment variables

  **Must NOT do**:
  - Do NOT include Supabase local — connect to remote Supabase
  - Do NOT optimize for production yet (use Dockerfile.dev, not Dockerfile)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None directly
  - **Blocked By**: T1 (needs directory structure)

  **References**:
  - Redis image: `redis:7-alpine`
  - Air hot reload: `cosmtrek/air`

  **Acceptance Criteria**:
  - [ ] `docker-compose up -d` starts both services
  - [ ] Redis accessible at `localhost:6379`
  - [ ] Go app container builds successfully

  **QA Scenarios**:
  ```
  Scenario: Docker Compose starts successfully
    Tool: Bash
    Steps:
      1. docker-compose up -d
      2. docker-compose ps
      3. redis-cli ping
    Expected Result: Both services show "running", Redis returns PONG
    Evidence: .sisyphus/evidence/task-02-docker.txt
  ```

  **Commit**: YES
  - Message: `feat(dev): docker compose local development`
  - Files: `docker-compose.yml`, `Dockerfile.dev`

- [ ] 3. sqlc Configuration + Connection Pool

  **What to do**:
  - Create `sqlc.yaml` configuration
  - Create `internal/db/queries/` directory for SQL files
  - Create `internal/db/db.go` with `pgx` connection pool setup
  - Configure connection pool settings (max open, max idle, lifetime)
  - Create pgvector custom type for sqlc (if needed, using `pgtype`)

  **Must NOT do**:
  - Do NOT write queries yet (that's T4)
  - Do NOT use GORM or other ORM

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T4 (needs sqlc.yaml)
  - **Blocked By**: T1 (directory structure)

  **References**:
  - Existing schema: `/Users/jorandob/Documents/Projects/SocialChef/socialchef-supabase/supabase/migrations/`
  - sqlc docs: https://docs.sqlc.dev/en/latest/reference/config.html
  - pgx driver: `github.com/jackc/pgx/v5`
  - pgvector: `github.com/pgvector/pgvector-go`

  **Acceptance Criteria**:
  - [ ] `sqlc.yaml` exists and is valid
  - [ ] `sqlc generate` runs without errors (even with empty queries)
  - [ ] Connection pool connects to Supabase Postgres

  **QA Scenarios**:
  ```
  Scenario: sqlc generates code
    Tool: Bash
    Steps:
      1. make sqlc-generate
      2. ls internal/db/generated/
    Expected Result: Generated files exist (db.go, models.go, etc.)
    Evidence: .sisyphus/evidence/task-03-sqlc.txt
  ```

  **Commit**: YES
  - Message: `feat(db): sqlc configuration and connection pool`
  - Files: `sqlc.yaml`, `internal/db/db.go`

- [ ] 4. Database Queries (sqlc)

  **What to do**:
  - Create SQL query files in `internal/db/queries/` for all operations:
    - `recipes.sql`: INSERT recipe, SELECT by ID, SELECT by user, UPDATE, DELETE
    - `ingredients.sql`: INSERT batch, SELECT by recipe
    - `instructions.sql`: INSERT batch, SELECT by recipe
    - `nutrition.sql`: INSERT, SELECT by recipe
    - `images.sql`: INSERT stored_image (with content_hash), INSERT recipe_image, SELECT by recipe
    - `social_owners.sql`: INSERT, SELECT by handle
    - `import_jobs.sql`: INSERT, UPDATE status/progress, SELECT by ID, SELECT by user
    - `profiles.sql`: SELECT by ID, UPDATE
    - `search.sql`: Hybrid search (text + vector similarity)
  - Run `sqlc generate` to create Go code
  - All user-scoped queries MUST include `WHERE created_by = $1` (service role RLS)

  **Must NOT do**:
  - Do NOT use ORM patterns — raw SQL only
  - Do NOT add queries for CMS tables (content_items, content_categories)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T14 (pipeline needs queries)
  - **Blocked By**: T3 (needs sqlc.yaml)

  **References**:
  - Existing schema: `/Users/jorandob/Documents/Projects/SocialChef/socialchef-supabase/supabase/migrations/`
  - Table structures: profiles, recipes, recipe_ingredients, recipe_instructions, recipe_nutrition, social_media_owners, stored_images, recipe_images, recipe_import_jobs
  - pgvector search: `SELECT ... ORDER BY embedding <=> $1 LIMIT $2`

  **Acceptance Criteria**:
  - [ ] All query files exist in `internal/db/queries/`
  - [ ] `sqlc generate` produces valid Go code
  - [ ] Generated `models.go` matches existing table structures

  **QA Scenarios**:
  ```
  Scenario: sqlc queries generate
    Tool: Bash
    Steps:
      1. make sqlc-generate
      2. grep -l "CreateRecipe" internal/db/*.go
    Expected Result: Generated files contain expected function names
    Evidence: .sisyphus/evidence/task-04-queries.txt
  ```

  **Commit**: YES
  - Message: `feat(db): add all sqlc queries`
  - Files: `internal/db/queries/*.sql`, generated `internal/db/*.go`

- [ ] 5. Configuration Management

  **What to do**:
  - Create `internal/config/config.go` with typed configuration struct
  - Load from environment variables using `os.Getenv` (no external lib needed)
  - Required vars:
    - `DATABASE_URL` — Supabase Postgres connection string
    - `SUPABASE_URL` — Supabase project URL
    - `SUPABASE_JWT_SECRET` — For JWT verification
    - `SUPABASE_SERVICE_ROLE_KEY` — For Storage/Realtime API calls
    - `REDIS_URL` — Redis connection for Asynq
    - `OPENAI_API_KEY` — OpenAI API
    - `GROQ_API_KEY` — Groq API
    - `APIFY_API_KEY` — Apify for TikTok
    - `PROXY_SERVER_URL`, `PROXY_API_KEY` — Instagram proxy
    - `OTEL_EXPORTER_OTLP_ENDPOINT` — OpenTelemetry
    - `PORT` — Server port (default 8080)
  - Add validation on startup (fail fast if missing required vars)

  **Must NOT do**:
  - Do NOT use Viper or other heavy config libraries — stdlib is enough
  - Do NOT add config file support — env vars only

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T7, T8, T9, T10 (all need config)
  - **Blocked By**: T1 (directory structure)

  **References**:
  - Go stdlib: `os.Getenv`, `os.LookupEnv`
  - Pattern: `config.Load()` called in `main.go`

  **Acceptance Criteria**:
  - [ ] `config.Load()` returns typed struct
  - [ ] Missing required vars causes clear error message
  - [ ] `.env.example` matches config struct fields

  **QA Scenarios**:
  ```
  Scenario: Config loads from env
    Tool: Bash
    Steps:
      1. export DATABASE_URL="postgres://test"
      2. export REDIS_URL="redis://localhost"
      3. go test ./internal/config/... -v
    Expected Result: Tests pass, config validates correctly
    Evidence: .sisyphus/evidence/task-05-config.txt
  ```

  **Commit**: YES
  - Message: `feat(config): configuration management`
  - Files: `internal/config/config.go`

- [ ] 6. OpenTelemetry Setup

  **What to do**:
  - Create `internal/telemetry/telemetry.go` with OTel initialization
  - Configure OTLP exporter for traces
  - Configure metrics exporter
  - Create helper functions:
    - `InitTelemetry(ctx, config) (shutdown func(), error)`
    - `Tracer(name) trace.Tracer`
    - `Meter(name) metric.Meter`
  - Add trace middleware for chi router
  - Integrate with Fly.io metrics endpoint

  **Must NOT do**:
  - Do NOT add logging abstraction — use `slog` from stdlib
  - Do NOT over-instrument — focus on HTTP handlers and worker tasks

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None directly, but used by all services
  - **Blocked By**: T1 (directory structure), T5 (config)

  **References**:
  - OTel Go SDK: `go.opentelemetry.io/otel`
  - OTLP exporter: `go.opentelemetry.io/otel/exporters/otlp/otlptrace`
  - Grafana Cloud OTel: uses OTLP over HTTP/gRPC

  **Acceptance Criteria**:
  - [ ] `InitTelemetry` initializes without error
  - [ ] HTTP requests are traced automatically
  - [ ] Traces export to configured endpoint

  **QA Scenarios**:
  ```
  Scenario: OTel initializes
    Tool: Bash
    Steps:
      1. export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
      2. go test ./internal/telemetry/... -v
    Expected Result: Tests pass, tracer is initialized
    Evidence: .sisyphus/evidence/task-06-otel.txt
  ```

  **Commit**: YES
  - Message: `feat(telemetry): OpenTelemetry setup`
  - Files: `internal/telemetry/telemetry.go`

- [ ] 7. Supabase JWT Auth Middleware

  **What to do**:
  - Create `internal/middleware/auth.go`
  - Implement JWT verification using `github.com/golang-jwt/jwt/v5`:
    - Extract `Authorization: Bearer <token>` header
    - Verify HMAC-SHA256 signature using `SUPABASE_JWT_SECRET`
    - Validate `iss` (issuer) matches Supabase URL
    - Validate `exp` (expiration)
    - Extract `sub` (user ID) and add to request context
  - Create `AuthMiddleware` chi middleware
  - Create helper `GetUserID(ctx) string`
  - Return 401 for missing/invalid tokens with JSON error response

  **Must NOT do**:
  - Do NOT just decode JWT — MUST verify signature (current TS bug)
  - Do NOT add role-based auth — not in current system

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T17-T20 (all API endpoints need auth)
  - **Blocked By**: T1 (directory), T5 (config for JWT secret)

  **References**:
  - Supabase JWT format: Uses HS256, issuer is Supabase URL
  - Current (buggy) implementation: `/supabase/functions/_shared/auth.ts` — just decodes, doesn't verify
  - JWT library: `github.com/golang-jwt/jwt/v5`

  **Acceptance Criteria**:
  - [ ] Valid Supabase JWT passes middleware
  - [ ] Missing token returns 401
  - [ ] Expired token returns 401
  - [ ] Invalid signature returns 401
  - [ ] User ID available via `GetUserID(ctx)`

  **QA Scenarios**:
  ```
  Scenario: Auth middleware validates JWT
    Tool: Bash
    Steps:
      1. go test ./internal/middleware/... -v -run TestAuthMiddleware
    Expected Result: All test cases pass (valid, missing, expired, invalid)
    Evidence: .sisyphus/evidence/task-07-auth.txt
  ```

  **Commit**: YES
  - Message: `feat(auth): JWT verification middleware`
  - Files: `internal/middleware/auth.go`, `internal/middleware/auth_test.go`

---

### Wave 2: Core Services (6 tasks — parallel after Wave 1)

- [ ] 8. Asynq Server + Client Setup

  **What to do**:
  - Create `internal/worker/server.go` with Asynq server initialization
  - Create `internal/worker/client.go` with Asynq client for enqueuing tasks
  - Define task types:
    - `task:process_recipe` — main pipeline
    - `task:cleanup_old_jobs` — scheduled cleanup
  - Create `cmd/worker/main.go` entrypoint
  - Configure Redis connection from config
  - Set up concurrency and queues

  **Must NOT do**:
  - Do NOT implement task handlers yet (T14)
  - Do NOT add priority queues — single queue is fine

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14, T16 (need Asynq setup)
  - **Blocked By**: T5 (config for Redis URL)

  **References**:
  - Asynq docs: `github.com/hibiken/asynq`
  - Pattern: `asynq.NewServer`, `asynq.NewClient`, `mux.HandleFunc`

  **Acceptance Criteria**:
  - [ ] Worker starts and connects to Redis
  - [ ] Client can enqueue tasks
  - [ ] Task types defined as constants

  **QA Scenarios**:
  ```
  Scenario: Asynq connects to Redis
    Tool: Bash
    Steps:
      1. redis-cli PING
      2. go run ./cmd/worker &
      3. sleep 2 && redis-cli KEYS '*'
    Expected Result: Worker starts, Redis shows Asynq keys
    Evidence: .sisyphus/evidence/task-08-asynq.txt
  ```

  **Commit**: YES
  - Message: `feat(worker): Asynq server and client setup`
  - Files: `internal/worker/server.go`, `internal/worker/client.go`, `cmd/worker/main.go`

- [ ] 9. OpenAI Client

  **What to do**:
  - Create `internal/services/openai/client.go`
  - Implement three methods:
    - `TranscribeAudio(ctx, audioURL) (string, error)` — uses `gpt-4o-mini-transcribe`
    - `ExtractRecipe(ctx, transcript, prompts) (*RecipeData, error)` — uses `gpt-3.5-turbo-1106`
    - `GenerateEmbedding(ctx, text) ([]float32, error)` — uses `text-embedding-ada-002`
  - Use stdlib `net/http` with retry wrapper
  - Copy prompts EXACTLY from existing code (do NOT rephrase)
  - Handle rate limits with exponential backoff

  **Must NOT do**:
  - Do NOT use OpenAI Go SDK — use raw HTTP (simpler, no deps)
  - Do NOT modify or "improve" the prompts

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14 (pipeline needs OpenAI)
  - **Blocked By**: T5 (config for API key)

  **References**:
  - Existing prompts: `/trigger/config/prompts.ts` — COPY VERBATIM
  - Model names: `gpt-4o-mini-transcribe`, `gpt-3.5-turbo-1106`, `text-embedding-ada-002`
  - Existing implementation: `/trigger/repositories/openaiRecipeAudioGenerationInfoRepository.ts`, `/trigger/repositories/openaiRecipeGenerationRepository.ts`

  **Acceptance Criteria**:
  - [ ] `TranscribeAudio` returns transcript string
  - [ ] `ExtractRecipe` returns structured recipe JSON
  - [ ] `GenerateEmbedding` returns 1536-dimension vector
  - [ ] Rate limits handled with retry

  **QA Scenarios**:
  ```
  Scenario: OpenAI transcription works
    Tool: Bash
    Steps:
      1. go test ./internal/services/openai/... -v -run TestTranscribeAudio
    Expected Result: Test passes with mock or real API
    Evidence: .sisyphus/evidence/task-09-openai.txt
  ```

  **Commit**: YES
  - Message: `feat(services): OpenAI client`
  - Files: `internal/services/openai/client.go`, `internal/services/openai/prompts.go`

- [ ] 10. Groq Client

  **What to do**:
  - Create `internal/services/groq/client.go`
  - Implement two methods:
    - `ValidateContent(ctx, transcript) (*ValidationResult, error)` — uses `openai/gpt-oss-20b`
    - `ExtractRecipe(ctx, transcript, prompts) (*RecipeData, error)` — uses `openai/gpt-oss-120b` (alternative to OpenAI)
  - Groq API is OpenAI-compatible — same request format, different base URL
  - Use stdlib `net/http` with retry wrapper
  - Copy prompts EXACTLY from existing code

  **Must NOT do**:
  - Do NOT use Groq SDK — raw HTTP with OpenAI-compatible format
  - Do NOT add Anthropic support — dead code

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14 (pipeline needs Groq for validation)
  - **Blocked By**: T5 (config for API key)

  **References**:
  - Groq API base: `https://api.groq.com/openai/v1`
  - Existing implementation: `/trigger/repositories/groqRecipeGenerationRepository.ts`, `/trigger/lib/contentValidator.ts`
  - Model names: `openai/gpt-oss-20b` (validation), `openai/gpt-oss-120b` (extraction)

  **Acceptance Criteria**:
  - [ ] `ValidateContent` returns isRecipe boolean
  - [ ] `ExtractRecipe` returns structured recipe JSON
  - [ ] Rate limits handled with retry

  **QA Scenarios**:
  ```
  Scenario: Groq validation works
    Tool: Bash
    Steps:
      1. go test ./internal/services/groq/... -v -run TestValidateContent
    Expected Result: Test passes
    Evidence: .sisyphus/evidence/task-10-groq.txt
  ```

  **Commit**: YES
  - Message: `feat(services): Groq client`
  - Files: `internal/services/groq/client.go`

- [ ] 11. Instagram Scraper

  **What to do**:
  - Create `internal/services/scraper/instagram.go`
  - Implement `FetchPost(ctx, url) (*InstagramPost, error)`
  - Use proxy server + Instagram GraphQL API (same as current system)
  - Extract: caption (transcript), video URL, images, owner handle
  - Handle errors gracefully (private posts, deleted, rate limited)
  - Retry logic for transient failures (3 attempts, exponential backoff)

  **Must NOT do**:
  - Do NOT add TikTok support here (separate T12)
  - Do NOT use Instagram SDK — raw HTTP via proxy

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14 (pipeline needs scraper)
  - **Blocked By**: T5 (config for proxy URL/key)

  **References**:
  - Existing implementation: `/trigger/repositories/instagramRepository.ts`
  - GraphQL query: `MediaDataQuery` from existing code
  - Proxy headers: `x-proxy-api-key`, `x-proxy-target-url`

  **Acceptance Criteria**:
  - [ ] `FetchPost` returns post data for valid Instagram URLs
  - [ ] Video URL extracted when present
  - [ ] Images extracted (multiple for carousels)
  - [ ] Errors returned for private/deleted posts

  **QA Scenarios**:
  ```
  Scenario: Instagram scraper fetches post
    Tool: Bash
    Steps:
      1. go test ./internal/services/scraper/... -v -run TestInstagramFetch
    Expected Result: Test passes with mock or real proxy
    Evidence: .sisyphus/evidence/task-11-instagram.txt
  ```

  **Commit**: YES
  - Message: `feat(services): Instagram scraper`
  - Files: `internal/services/scraper/instagram.go`

- [ ] 12. TikTok Scraper

  **What to do**:
  - Create `internal/services/scraper/tiktok.go`
  - Implement `FetchVideo(ctx, url) (*TikTokVideo, error)`
  - Use Apify actor `GdWCkxBtKWOsKjdch` (same as current system)
  - Extract: description (transcript), video URL, author handle
  - Poll Apify for task completion (can take 1-3 minutes)
  - Handle timeouts gracefully

  **Must NOT do**:
  - Do NOT add Instagram support here (T11)
  - Do NOT use TikTok API directly — must use Apify

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14 (pipeline needs scraper)
  - **Blocked By**: T5 (config for Apify key)

  **References**:
  - Existing implementation: `/trigger/repositories/tiktokRepository.ts`
  - Apify API: `https://api.apify.com/v2/acts/` + actor ID
  - Timeout: 3 minutes max

  **Acceptance Criteria**:
  - [ ] `FetchVideo` returns video data for valid TikTok URLs
  - [ ] Video URL extracted
  - [ ] Description extracted
  - [ ] Timeout handled gracefully

  **QA Scenarios**:
  ```
  Scenario: TikTok scraper fetches video
    Tool: Bash
    Steps:
      1. go test ./internal/services/scraper/... -v -run TestTikTokFetch
    Expected Result: Test passes with mock or real Apify
    Evidence: .sisyphus/evidence/task-12-tiktok.txt
  ```

  **Commit**: YES
  - Message: `feat(services): TikTok scraper`
  - Files: `internal/services/scraper/tiktok.go`

- [ ] 13. Supabase Storage Client

  **What to do**:
  - Create `internal/services/storage/client.go`
  - Implement methods:
    - `UploadImage(ctx, bucket, path, imageData) (url string, error)`
    - `GetPublicURL(bucket, path) string`
  - Use Supabase Storage REST API (not SDK)
  - Handle upload errors (bucket not found, size limit)

  **Must NOT do**:
  - Do NOT use Supabase Go SDK — raw HTTP is simpler
  - Do NOT add file validation beyond what exists

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: T14 (pipeline needs storage)
  - **Blocked By**: T5 (config for Supabase URL/key)

  **References**:
  - Supabase Storage API: `POST /storage/v1/object/{bucket}/{path}`
  - Auth: `Authorization: Bearer {service_role_key}`
  - Buckets: `recipes`, `profiles`

  **Acceptance Criteria**:
  - [ ] `UploadImage` uploads and returns public URL
  - [ ] `GetPublicURL` returns correct URL format
  - [ ] Errors handled gracefully

  **QA Scenarios**:
  ```
  Scenario: Storage upload works
    Tool: Bash
    Steps:
      1. go test ./internal/services/storage/... -v -run TestUploadImage
    Expected Result: Test passes
    Evidence: .sisyphus/evidence/task-13-storage.txt
  ```

  **Commit**: YES
  - Message: `feat(services): Supabase Storage client`
  - Files: `internal/services/storage/client.go`

---

### Wave 3: Business Logic (6 tasks — parallel after Wave 2)

- [ ] 14. Recipe Processing Pipeline (Asynq Handler)

  **What to do**:
  - Create `internal/worker/handlers/process_recipe.go`
  - Implement full pipeline as Asynq task handler:
    1. **Detect origin** from URL (Instagram vs TikTok)
    2. **Fetch content** using appropriate scraper (T11/T12)
    3. **Transcribe audio** using OpenAI `gpt-4o-mini-transcribe` (T9)
    4. **Validate content** using Groq `openai/gpt-oss-20b` (T10)
    5. **Extract recipe** using OpenAI `gpt-3.5-turbo-1106` or Groq (T9/T10)
    6. **Generate embedding** using OpenAI `text-embedding-ada-002` (T9)
    7. **Save transactionally** using sqlc queries (T4)
    8. **Broadcast progress** at each step via Realtime (T15)
  - Update `recipe_import_jobs` status at each step
  - Handle errors with structured `TaskErrorDetails`
  - Copy prompts EXACTLY from `/trigger/config/prompts.ts`

  **Must NOT do**:
  - Do NOT skip validation step
  - Do NOT save partial data on error — use transaction rollback
  - Do NOT modify prompts

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: NO — depends on all Wave 2 tasks
  - **Parallel Group**: Wave 3 (after Wave 2)
  - **Blocks**: T17 (recipe endpoint enqueues this task)
  - **Blocked By**: T4, T8, T9, T10, T11, T12, T13

  **References**:
  - Existing pipeline: `/trigger/tasks/recipe/processRecipe.ts`
  - Prompts: `/trigger/config/prompts.ts` — COPY VERBATIM
  - Error format: `TaskErrorDetails` with `errorType`, `errorCode`, `step`, `message`, etc.

  **Acceptance Criteria**:
  - [ ] Pipeline processes Instagram URLs end-to-end
  - [ ] Pipeline processes TikTok URLs end-to-end
  - [ ] Progress updates broadcast to Realtime
  - [ ] Recipe saved transactionally (all or nothing)
  - [ ] Errors return structured `TaskErrorDetails`

  **QA Scenarios**:
  ```
  Scenario: Pipeline processes Instagram URL
    Tool: Bash
    Steps:
      1. Enqueue task with test Instagram URL
      2. Monitor recipe_import_jobs table for status changes
      3. Query recipes table for new record
    Expected Result: Recipe appears in database with all related data
    Evidence: .sisyphus/evidence/task-14-pipeline-instagram.txt

  Scenario: Pipeline handles invalid URL gracefully
    Tool: Bash
    Steps:
      1. Enqueue task with invalid URL
      2. Check job status in recipe_import_jobs
    Expected Result: Job status shows error with TaskErrorDetails
    Evidence: .sisyphus/evidence/task-14-pipeline-error.txt
  ```

  **Commit**: YES
  - Message: `feat(worker): recipe processing pipeline`
  - Files: `internal/worker/handlers/process_recipe.go`

- [ ] 15. Realtime Progress Broadcaster

  **What to do**:
  - Create `internal/services/realtime/broadcaster.go`
  - Implement `BroadcastProgress(ctx, jobID, step, status, payload)`
  - Use Supabase Realtime REST API: `POST /realtime/v1/api/broadcast`
  - Channel format: `recipe:{jobID}`
  - Event: `status_update`
  - Payload matches existing `updatePayload` shape from TypeScript

  **Must NOT do**:
  - Do NOT use WebSocket server — REST API broadcast only
  - Do NOT change event/payload format — must match current system

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 3
  - **Blocks**: T14 (pipeline needs broadcaster)
  - **Blocked By**: T5 (config for Supabase URL/key)

  **References**:
  - Existing broadcast format: `recipe-bridge/index.ts:170-180`
  - Supabase Realtime REST: `POST /realtime/v1/api/broadcast`

  **Acceptance Criteria**:
  - [ ] `BroadcastProgress` sends to correct channel
  - [ ] Payload format matches TypeScript implementation
  - [ ] Errors handled gracefully (don't fail pipeline)

  **QA Scenarios**:
  ```
  Scenario: Progress broadcast works
    Tool: Bash
    Steps:
      1. Call BroadcastProgress with test job ID
      2. Subscribe to channel in Supabase Realtime
    Expected Result: Event received with correct payload shape
    Evidence: .sisyphus/evidence/task-15-realtime.txt
  ```

  **Commit**: YES
  - Message: `feat(services): Realtime progress broadcaster`
  - Files: `internal/services/realtime/broadcaster.go`

- [ ] 16. Cleanup Cron Job

  **What to do**:
  - Create `internal/worker/handlers/cleanup.go`
  - Implement `cleanup_old_import_jobs` task handler
  - Call `supabase.rpc('cleanup_old_import_jobs')` via Postgres
  - Register with Asynq scheduler for daily execution at 2 AM UTC
  - Log cleanup results

  **Must NOT do**:
  - Do NOT implement cleanup logic in Go — use existing RPC function

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: T8 (needs Asynq scheduler)

  **References**:
  - Existing cron: `/trigger/tasks/cleanupOldJobs.ts`
  - Asynq scheduler: `asynq.NewScheduler`, `scheduler.Register`

  **Acceptance Criteria**:
  - [ ] Task runs daily at 2 AM UTC
  - [ ] Old import jobs are cleaned up
  - [ ] Results logged

  **QA Scenarios**:
  ```
  Scenario: Cleanup task runs
    Tool: Bash
    Steps:
      1. Manually trigger cleanup task
      2. Check logs for cleanup count
    Expected Result: Task completes without error, old jobs removed
    Evidence: .sisyphus/evidence/task-16-cleanup.txt
  ```

  **Commit**: YES
  - Message: `feat(worker): cleanup cron job`
  - Files: `internal/worker/handlers/cleanup.go`

- [ ] 17. POST /recipe Endpoint

  **What to do**:
  - Create `internal/api/handlers/recipe.go`
  - Implement `POST /recipe` handler:
    - Validate request body: `{url: string, generationModel?: string}`
    - Detect origin from URL
    - Create `recipe_import_jobs` record
    - Enqueue `task:process_recipe` with Asynq client
    - Return response: `{success: true, jobId: string, message: string, requestId: string}`
  - Apply auth middleware
  - Add request validation

  **Must NOT do**:
  - Do NOT change request/response format from TypeScript
  - Do NOT block on processing — must return immediately with jobId

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1 and T14)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: T7 (auth), T8 (Asynq client), T14 (task exists)

  **References**:
  - Existing endpoint: `/supabase/functions/recipe/index.ts`
  - Response shape: `{success, jobId, message, requestId}`

  **Acceptance Criteria**:
  - [ ] Returns 202 Accepted with jobId
  - [ ] Creates import job in database
  - [ ] Enqueues Asynq task
  - [ ] Rejects invalid URLs

  **QA Scenarios**:
  ```
  Scenario: Recipe submission works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/recipe -H "Authorization: Bearer $JWT" -d '{"url":"https://www.instagram.com/p/test/"}'
    Expected Result: HTTP 202, response contains {"success":true,"jobId":"..."}
    Evidence: .sisyphus/evidence/task-17-recipe-endpoint.txt

  Scenario: Missing auth returns 401
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/recipe -d '{"url":"https://test.com"}'
    Expected Result: HTTP 401
    Evidence: .sisyphus/evidence/task-17-recipe-auth.txt
  ```

  **Commit**: YES
  - Message: `feat(api): POST /recipe endpoint`
  - Files: `internal/api/handlers/recipe.go`

- [ ] 18. POST /recipe-status Endpoint

  **What to do**:
  - Create `internal/api/handlers/recipe_status.go`
  - Implement `POST /recipe-status` handler:
    - Validate request: `{jobId: string}`
    - Query `recipe_import_jobs` table for status
    - Return: `{status: string, isCompleted: bool, error?: TaskErrorDetails}`
  - Apply auth middleware
  - User can only query their own jobs

  **Must NOT do**:
  - Do NOT query Trigger.dev — use local database
  - Do NOT change response format

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: T4 (queries), T7 (auth)

  **References**:
  - Existing endpoint: `/supabase/functions/recipe-status/index.ts`
  - Query `recipe_import_jobs` table directly

  **Acceptance Criteria**:
  - [ ] Returns job status from database
  - [ ] Returns 404 if job not found
  - [ ] Returns 403 if job belongs to different user

  **QA Scenarios**:
  ```
  Scenario: Recipe status works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/recipe-status -H "Authorization: Bearer $JWT" -d '{"jobId":"..."}'
    Expected Result: HTTP 200, response contains {"status":"...","isCompleted":...}
    Evidence: .sisyphus/evidence/task-18-status.txt
  ```

  **Commit**: YES
  - Message: `feat(api): POST /recipe-status endpoint`
  - Files: `internal/api/handlers/recipe_status.go`

- [ ] 19. GET /user-import-status Endpoint

  **What to do**:
  - Create `internal/api/handlers/user_import_status.go`
  - Implement `GET /user-import-status` handler:
    - Get user ID from auth context
    - Query `recipe_import_jobs` for user's jobs
    - Return: `{data: [{jobId, status, ...}]}`
  - Apply auth middleware
  - Support pagination (optional, match current system)

  **Must NOT do**:
  - Do NOT query Trigger.dev — use local database

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: T4 (queries), T7 (auth)

  **References**:
  - Existing endpoint: `/supabase/functions/user-import-status/index.ts`

  **Acceptance Criteria**:
  - [ ] Returns list of user's import jobs
  - [ ] Only returns jobs for authenticated user

  **QA Scenarios**:
  ```
  Scenario: User import status works
    Tool: Bash (curl)
    Steps:
      1. curl http://localhost:8080/user-import-status -H "Authorization: Bearer $JWT"
    Expected Result: HTTP 200, response contains {"data":[...]}
    Evidence: .sisyphus/evidence/task-19-user-status.txt
  ```

  **Commit**: YES
  - Message: `feat(api): GET /user-import-status endpoint`
  - Files: `internal/api/handlers/user_import_status.go`

---

### Wave 4: Finalization (5 tasks — parallel after Wave 3)

- [ ] 20. POST /generate-embedding Endpoint

  **What to do**:
  - Create `internal/api/handlers/generate_embedding.go`
  - Implement `POST /generate-embedding` handler:
    - Validate request: `{text: string}`
    - Call OpenAI `text-embedding-ada-002`
    - Return: `{embedding: float[]}`
  - Apply auth middleware

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 2)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: T7 (auth), T9 (OpenAI client)

  **References**:
  - Existing endpoint: `/supabase/functions/generate-embedding/index.ts`
  - Model: `text-embedding-ada-002`

  **Acceptance Criteria**:
  - [ ] Returns 1536-dimension embedding vector
  - [ ] Handles errors gracefully

  **QA Scenarios**:
  ```
  Scenario: Embedding generation works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/generate-embedding -H "Authorization: Bearer $JWT" -d '{"text":"test recipe"}'
    Expected Result: HTTP 200, response contains {"embedding":[...]}
    Evidence: .sisyphus/evidence/task-20-embedding.txt
  ```

  **Commit**: YES
  - Message: `feat(api): POST /generate-embedding endpoint`
  - Files: `internal/api/handlers/generate_embedding.go`

- [ ] 21. Integration Tests

  **What to do**:
  - Create `internal/integration/` directory
  - Write integration tests for:
    - Full recipe pipeline (Instagram and TikTok)
    - API endpoints (all 4)
    - Auth flow
    - Realtime broadcasting
  - Use test database connection
  - Create `Makefile` target: `make test-integration`

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: All previous tasks

  **Acceptance Criteria**:
  - [ ] All integration tests pass
  - [ ] Coverage includes happy path and error scenarios

  **Commit**: YES
  - Message: `test: integration tests`
  - Files: `internal/integration/*.go`

- [ ] 22. Fly.io Deployment

  **What to do**:
  - Create production `Dockerfile` (multi-stage build)
  - Create `fly.toml` with:
    - Two processes: `server` (API), `worker` (Asynq)
    - Redis attachment (Fly Redis or Upstash)
    - Environment variables from secrets
    - Health checks
  - Create `Makefile` target: `make deploy`
  - Document deployment process

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4
  - **Blocks**: T24
  - **Blocked By**: T1, T2 (Docker experience)

  **References**:
  - Fly.io docs: Multi-process apps
  - Redis on Fly: `fly redis attach` or Upstash

  **Acceptance Criteria**:
  - [ ] `fly deploy` succeeds
  - [ ] Both processes running
  - [ ] Health check passes

  **QA Scenarios**:
  ```
  Scenario: Fly.io deployment works
    Tool: Bash
    Steps:
      1. make deploy
      2. fly status
      3. curl https://APP.fly.dev/health
    Expected Result: All processes running, health endpoint returns 200
    Evidence: .sisyphus/evidence/task-22-deploy.txt
  ```

  **Commit**: YES
  - Message: `feat(deploy): Fly.io configuration`
  - Files: `Dockerfile`, `fly.toml`

- [ ] 23. Production Config + Secrets

  **What to do**:
  - Set Fly.io secrets for all environment variables:
    - `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_JWT_SECRET`
    - `SUPABASE_SERVICE_ROLE_KEY`, `REDIS_URL`
    - `OPENAI_API_KEY`, `GROQ_API_KEY`, `APIFY_API_KEY`
    - `PROXY_SERVER_URL`, `PROXY_API_KEY`
    - `OTEL_EXPORTER_OTLP_ENDPOINT`
  - Create `fly secrets set` commands in documentation
  - Verify secrets are not logged

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4
  - **Blocks**: T24
  - **Blocked By**: None

  **Acceptance Criteria**:
  - [ ] All secrets set in Fly.io
  - [ ] App starts successfully with secrets
  - [ ] Secrets not exposed in logs

  **Commit**: NO (secrets are external)

- [ ] 24. End-to-End Verification

  **What to do**:
  - Deploy to Fly.io staging environment
  - Run full E2E tests:
    - Submit real Instagram URL → verify recipe appears
    - Submit real TikTok URL → verify recipe appears
    - Verify Realtime broadcasts
    - Verify OTel traces in Grafana
  - Compare Go API responses to TypeScript API responses
  - Document any differences

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3 and T22, T23)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: T22, T23

  **Acceptance Criteria**:
  - [ ] Instagram URL processes end-to-end on production
  - [ ] TikTok URL processes end-to-end on production
  - [ ] API responses match TypeScript implementation
  - [ ] OTel traces visible in Grafana

  **QA Scenarios**:
  ```
  Scenario: Production E2E works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST https://APP.fly.dev/recipe -H "Authorization: Bearer $JWT" -d '{"url":"https://www.instagram.com/p/REAL_POST/"}'
      2. Poll /recipe-status until completed
      3. Query Supabase for recipe data
    Expected Result: Recipe appears in database with correct data
    Evidence: .sisyphus/evidence/task-24-e2e.txt
  ```

  **Commit**: YES
  - Message: `docs: E2E verification complete`
  - Files: `docs/deployment.md`

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — Verify all deliverables exist and work
- [ ] F2. **API Contract Verification** — Compare Go responses to TypeScript responses
- [ ] F3. **End-to-End QA** — Submit real Instagram/TikTok URLs, verify complete flow
- [ ] F4. **Deployment Verification** — Confirm Fly.io app is live and healthy

---

## Commit Strategy

Commit after each wave completes and all tests pass:
- `feat(scaffold): project structure and local dev setup`
- `feat(db): sqlc queries and connection pool`
- `feat(auth): JWT verification middleware`
- `feat(services): AI clients and scrapers`
- `feat(worker): recipe processing pipeline`
- `feat(api): HTTP endpoints`
- `feat(deploy): Fly.io configuration`
- `test: integration tests`

---

## Success Criteria

### Verification Commands
```bash
# Local dev starts
docker-compose up -d && curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Submit recipe
curl -X POST http://localhost:8080/recipe \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.instagram.com/p/EXAMPLE/"}'
# Expected: {"success":true,"jobId":"...","message":"..."}

# Check job status  
curl -X POST http://localhost:8080/recipe-status \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"jobId":"..."}'
# Expected: {"status":"completed","isCompleted":true}

# All tests pass
go test ./... -v
# Expected: PASS
```

### Final Checklist
- [ ] All 4 endpoints return correct response shapes
- [ ] Instagram URLs process end-to-end
- [ ] TikTok URLs process end-to-end
- [ ] Realtime progress updates broadcast
- [ ] JWT auth rejects invalid tokens
- [ ] Recipe saves are transactional
- [ ] Daily cleanup job runs
- [ ] Deployed to Fly.io
- [ ] OTel traces visible in Grafana
