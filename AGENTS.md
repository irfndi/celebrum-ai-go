# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/main.go` is the HTTP entrypoint; runtime wiring lives under `internal/` (API handlers, services, config, telemetry, cache, CCXT integration).
- Shared packages live in `pkg/`; prefer `internal/` for app-specific logic and keep reusable abstractions in `pkg/`.
- The Bun-based exchange service resides in `ccxt-service/`; treat it as a sibling project with its own tests and Docker build.
- Configuration templates are in `configs/`; copy `.env.template` or `configs/config.template.yml` when bootstrapping environments.
- Tests live beside source as `*_test.go`; integration suites and mocks live in `test/`.

## Build, Test, and Development Commands
- `make build` – compile the Go API to `bin/celebrum-ai`.
- `make run` – build then run the API locally.
- `make dev` – hot reload via `air`; requires `go install github.com/air-verse/air@latest`.
- `make test` – run Go unit tests and Bun tests (if Bun is installed).
- `make lint` / `make typecheck` – run `golangci-lint` and `go vet`.
- `make dev-setup` / `make dev-down` – start/stop Postgres + Redis via Docker Compose.
- `make coverage-check` – run the non-blocking coverage gate (default threshold 80%).

## Coding Style & Naming Conventions
- Target Go 1.25; format with `gofmt`/`goimports` (tabs, canonical imports).
- Use idiomatic Go naming (exported identifiers in `CamelCase`, packages lowercase).
- For Bun/TypeScript, rely on `bunx` formatters (`bun run format`) and keep strict lint output clean.
- Prefer short, descriptive directory and file names; keep shared interfaces in `pkg/`.

## Testing Guidelines
- Use table-driven tests for Go packages; place mocks in `internal/testutil` or `test/testmocks`.
- Run `make test` before pushing; for targeted Go packages use `go test ./internal/services`.
- Coverage expectations: ≥80% on touched packages; upload artifacts via CI (`ci-artifacts/coverage`).
- Name integration tests under `test/integration` and skip external dependencies unless orchestrated in CI.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (`feat:`, `fix:`, `refactor:`, etc.); keep subjects imperative and ≤72 chars.
- Reference issues in commit bodies or PR descriptions where applicable.
- PRs must include: summary of changes, rationale, test evidence (`make test` output), and updated docs/config as needed.
- Ensure CI (build, lint, tests, Trivy scan) passes before requesting review.

## Security & Configuration Tips
- Never commit secrets; derive local env files from `.env.template`.
- Use `make security` for gosec, gitleaks, and Docker checks prior to releases.
- Gate external webhooks via the provided scripts (`scripts/setup-github-secrets.sh`, `webhook-control.sh`).
