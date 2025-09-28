# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/main.go` — application entrypoint (HTTP/API server).
- `internal/{api,handlers,services,models,database,middleware,telemetry,ccxt,utils}` — core app layers.
- `pkg/{ccxt,interfaces,utils}` — shared/stable packages and interfaces.
- `ccxt-service/` — Bun-based exchange service (Node/Bun runtime).
- `test/{integration,benchmark,testmocks}` — integration, benchmarks, and mocks; unit tests live next to source as `*_test.go`.
- `configs/, config.yaml`, `.env*` — configuration; copy from `.env.example` for local use.
- `scripts/, database/, migrations/`, `docs/` — ops scripts, DB tooling, and documentation.

## Build, Test, and Development Commands
- `make build` — compile to `bin/celebrum-ai`; `make run` runs it.
- `make dev` — hot reload (requires `air`).
- `make dev-setup` / `make dev-down` — start/stop Postgres + Redis via Docker.
- `make test` — run Go tests (and ccxt tests when Bun is available).
- `make test-coverage` — generate `coverage.html`.
- `make lint` `make typecheck` — run `golangci-lint` and `go vet`.
- `make fmt` / `make fmt-check` — format and verify formatting.
- `make docker-build` / `make docker-run` — containerize and run with Compose.
- `make migrate` or `make auto-migrate` — database migrations.

## Coding Style & Naming Conventions
- Language: Go 1.25. Format with `gofmt`/`goimports` (tabs, canonical imports).
- Lint with `golangci-lint`; fix warnings or justify in code review.
- Packages: short, lowercase names; exported identifiers use `CamelCase`.
- Tests: table-driven where appropriate; avoid flaky time/network dependencies.
- Optional: install hooks `pre-commit install` to run format/lint locally.

## Testing Guidelines
- Unit tests colocated: `*_test.go`; integration tests in `test/integration`.
- Use `make test` for all suites; prefer deterministic tests with mocks from `internal/testutil` and `test/testmocks`.
- Aim for meaningful coverage on changed code; verify with `make test-coverage`.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (seen in history): `feat:`, `fix:`, `refactor:`, `test:`, `ci:`.
- Commit message: short imperative subject; optional body explains why and how.
- PRs must include: clear description, rationale, testing notes (`make test` output), and updated docs/config if applicable. Link issues.
- All CI-style checks (`ci-*` targets) should pass before review.

## Security & Configuration Tips
- Never commit secrets; use `.env` derived from `.env.example` and `config.yaml` overrides.
- Run `make security` to scan (gosec, gitleaks, basic Docker checks).
- External webhooks: gate via `make webhook-enable`/`webhook-disable` in controlled environments only.
