
## 2025-02-24: Fly.io Multi-Process Deployment

### Pattern: Fly.io Multi-Process Configuration
- Use `[processes]` section in fly.toml to define multiple processes
- Reference specific processes in `[[http_service]]` with `processes = ["server"]`
- Each process runs the command defined in the Dockerfile's CMD or explicitly in the process definition
- Format: `process_name = "command_to_run"`

### E2E Verification Script Pattern
- Accept both positional args and environment variables for flexibility
- Use curl with `-w "\n%{http_code}"` to capture both body and status code
- Implement polling with configurable timeout for async operations
- Provide colored output for better readability (RED/GREEN/YELLOW)
- Structure tests as separate functions for maintainability

### Makefile Integration
- Add new targets to `.PHONY` declaration
- Pass variables through $(VARNAME) syntax to scripts
- Keep targets simple and delegate to scripts for complex logic

### Fly.io Process Configuration
```toml
[processes]
  server = "/app/server"
  worker = "/app/worker"

[http_service]
  processes = ["server"]  # Only server handles HTTP
```

### Key Files Modified
- `fly.toml`: Added processes section and linked http_service to server process
- `scripts/verify-e2e.sh`: Created comprehensive E2E test suite
- `Makefile`: Added verify-e2e target
