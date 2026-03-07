# SocialChef Remy API - Bruno Collection

This Bruno collection provides automated API testing for the SocialChef Remy recipe extraction service.

## Prerequisites

1. Install Bruno CLI:
   ```bash
   npm install -g @usebruno/cli
   ```

2. Copy environment template and fill in your values:
   ```bash
   cp ../.env.bruno.example .env.bruno
   ```
   
   Edit `.env.bruno` and add:
   - `SUPABASE_URL` - Your Supabase project URL
   - `SUPABASE_JWT_SECRET` - Your JWT secret for token generation

## Usage

### Run All Requests (Local)
```bash
bru run --env local
```

### Run All Requests (Production)
```bash
bru run --env fly
```

### Run Specific Folder
```bash
bru run --env local --folder "1-Recipe"
```

## Collection Structure

```
bruno/
├── 0-Health/           # Health check (no auth)
├── 1-Recipe/           # Recipe import endpoints
├── 2-Embedding/        # Embedding generation
├── 3-Search/           # Search endpoints (not exposed yet)
├── environments/
│   ├── local.bru      # Local development
│   └── fly.bru        # Production (fly.io)
└── collection.bru     # JWT auth script
```

## Authentication

This collection uses **automatic JWT token generation**:

- Tokens are generated using `SUPABASE_JWT_SECRET` from environment
- Tokens cached for 1 hour (with 1-minute safety buffer)
- Tokens automatically refreshed when expired
- 401 responses trigger token regeneration

No manual token management required!

## Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Health check |
| `/api/recipe` | POST | Yes | Import recipe from URL |
| `/api/recipe-status` | GET | Yes | Check import job status |
| `/api/user-import-status` | GET | Yes | List user import jobs |
| `/api/generate-embedding` | POST | Yes | Queue embedding generation |
| `/api/search` | POST | Yes | Hybrid search *(not exposed)* |
| `/api/search-semantic` | POST | Yes | Semantic search *(not exposed)* |
| `/api/search-by-name` | POST | Yes | Name search *(not exposed)* |

## Troubleshooting

### "Missing required environment variables"
Create `.env.bruno` file from template with your Supabase credentials.

### 401 Unauthorized
Token may be invalid. The collection will auto-refresh on next request.

### 404 on search endpoints
These endpoints exist in code but routes aren't registered yet. Update `cmd/server/main.go` to expose them.

## Notes

- Health check is public (no JWT required)
- All other endpoints require valid JWT token
- Search endpoints are prepared but not exposed in current router config


## Known Issues

### Async JWT Generation in CLI
**Problem:** Bruno CLI doesn't wait for async crypto operations in pre-request scripts. The request is sent before the JWT is generated.

**Solution:** Add a pre-generated JWT token to your environment:

1. Generate a JWT using your `SUPABASE_JWT_SECRET`:
   ```bash
   # Use a JWT generator or https://jwt.io
   # Payload should be:
   # {
   #   "sub": "test-user-uuid-1234-5678-9012-345678901234",
   #   "iss": "YOUR_SUPABASE_URL/auth/v1",
   #   "exp": 1234567890
   # }
   ```

2. Add the token to `bruno/environments/local.bru`:
   ```
   vars {
     baseUrl: http://localhost:8080
     testUserId: test-user-uuid-1234-5678-9012-345678901234
     jwtToken: eyJhbGciOiJIUzI1NiIs...
   }
   ```

**Alternative:** Use Bruno Desktop App, which handles async operations better than CLI.
