# AGENTS.md

Go CLI for the [Nordic Financial News API](https://nordicfinancialnews.com). Binary name: `nfn`.

## Build & Test

```bash
go build ./...                    # Build
go test -race ./...               # All tests (with race detector)
go test -race ./internal/cmd/...  # Test one package
go vet ./...                      # Vet
```

CI also runs `golangci-lint` and `govulncheck`. Releases are cut via GoReleaser on `v*` tag push and distributed through Homebrew (`nordic-financial-news/tap/nfn`).

## Architecture

```
cmd/nfn/main.go          → Entrypoint, calls cmd.Execute()
internal/cmd/             → Cobra commands (root, articles, stories, companies, etc.)
internal/api/             → HTTP client, typed API methods, domain structs
internal/config/          → Viper config + keyring-based API key storage
internal/output/          → Table/JSON formatter, JSON envelope support
```

### How it fits together

1. `root.go`'s `PersistentPreRunE` initializes config, detects TTY, creates the API client and formatter, and injects both into the command context.
2. Every command pulls its dependencies from context via `apiClientFromContext(cmd)` and `formatterFromContext(cmd)`.
3. All commands follow the same pattern: get client + formatter → call API → render output.

### Command structure

Each API resource gets its own file in `internal/cmd/` (e.g., `articles.go`, `companies.go`). Each file defines a parent command and subcommands (`list`, `get`, etc.), registers them in `init()`, and wires up flags. The `commands.go` file maintains a catalog used by `nfn commands`.

## Key Patterns

### Auth

API key is resolved in order: `NFN_API_KEY` env var → system keyring → error. Commands that don't need auth (doctor, version, completion) are annotated with `"skipAuth": "true"` — the root `PersistentPreRunE` checks this and skips client setup.

### Output

TTY auto-detection: interactive terminal → table, piped → JSON. Override with `--format table` or `--format json`. All JSON responses use an envelope: `{ok, data, summary, breadcrumbs}`. Errors use `{ok: false, error, data: null}`.

### Pagination

Cursor-based. `--all` auto-paginates using generic helpers `ListPage[T]()` and `ListAll[T]()` in `internal/api/`. `--limit` caps per-page results, `--cursor` continues from a previous `next_cursor`.

### API client

Single `Get()` method for all endpoints. Automatically injects auth header, user-agent, and accept header. Auto-retries HTTP 429 once using the `Retry-After` header (max 60s). HTTPS-only enforcement via `resolveBaseURL()`. Path parameters escaped with `url.PathEscape()`.

### Breadcrumbs

`related.go` maps command paths to suggested follow-up commands. These appear in the `breadcrumbs` array in JSON envelope output, helping agents and scripts discover what to do next.

## Adding a New API Endpoint

1. **`internal/api/`** — Add or update structs and client methods matching the API shape. Use the existing generic `ListPage[T]`/`ListAll[T]` helpers for paginated endpoints.
2. **`internal/cmd/`** — Add a Cobra command file with flags, examples, and help text. Follow the context-based pattern: extract client/formatter, call API, render.
3. **`internal/cmd/commands.go`** — Register the command in the `nfn commands` catalog.
4. **`internal/cmd/related.go`** — Add breadcrumb entries for the new command.
5. **Tests** — Add tests for both the API layer (mock HTTP server) and the command layer.

## Testing Conventions

- Tests live in `_test.go` files in the same package (not a separate `_test` package).
- API tests use `httptest.NewServer` to mock HTTP responses — no external mocking libraries.
- Table-driven tests with `t.Run()` subtests are the standard pattern.
- Use `t.Parallel()` on most tests, but avoid it when touching shared state:
  - `rootCmd.Commands()` sorts in place, so command-tree tests can't run in parallel.
  - The global `Version` variable must be saved/restored if modified.
- `t.Setenv()` for temporary environment variable overrides.

## Conventions

- **Ticker format in examples**: use short form `VOLV-B`, never `VOLV-B.ST` or `VOLV-B:XSTO`.
- **Company identifier**: commands that take a company accept either a ticker or company ID — not all companies have tickers.
- **Flag-to-param mapping**: CLI flags use hyphens (`--content-type`), API params use underscores (`content_type`). Boolean flags become string `"true"` in URL params.
- **Config**: lives at `~/.config/nfn/config.yaml`, managed by Viper with `NFN_` env prefix.
- **Error format**: API errors follow RFC 9457 problem detail format. Error response bodies are capped at 4096 bytes.
- **Dependencies**: kept minimal — cobra, viper, keyring, go-pretty, term. Avoid adding new dependencies without good reason.
