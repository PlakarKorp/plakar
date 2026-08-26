# AGENTS.md

Instructions for AI coding agents working in this repo (Claude Code, Cursor, Copilot, Codex,
Aider, Zed, Gemini CLI, ...). Follows the [agents.md](https://agents.md) standard.
Humans: `README.md` for usage, `CONTRIBUTING.md` for process, `SECURITY.md` for disclosures.

## Working rules (read first)

- **Smallest change that satisfies the request.** No drive-by refactors, dependency bumps, or
  Go toolchain bumps unless that is the explicit task.
- **Storage-engine and archive-format logic lives in the separate `github.com/PlakarKorp/kloset`
  module, not here.** This repo is the CLI, server, API, and UI wiring on top of Kloset. If a
  change belongs in the engine, say so instead of reimplementing it here.
- **This is backup + encryption software.** Never weaken encryption/crypto defaults; never log
  keys, secrets, passphrases, or plaintext user data. Treat data-integrity paths with extra care.
- **Before you call a change done:** `go build ./...`, `go vet ./...`, and `make test` all pass,
  and you added or updated tests for any behavior change.
- **Don't commit, push, open PRs, or create branches** unless explicitly asked.

## Project

`plakar` is an open source backup engine: encrypted, deduplicated, verifiable, scalable.
Data is deduplicated, compressed, and encrypted at the source before it leaves the perimeter.

- **Module:** `github.com/PlakarKorp/plakar` · **License:** ISC · **Entry:** `main.go` (binary `plakar`)
- **Go:** toolchain pinned in `go.mod` (currently 1.25; builds on 1.23.3+)
- **Engine (separate module):** `github.com/PlakarKorp/kloset`, the immutable content-addressed store
- **Data sources/backends (separate modules):** `github.com/PlakarKorp/integrations/*`

## Commands

```bash
go build -v .        # or `make` -> ./plakar
make test            # go test ./...
make coverage        # total coverage, testing/ helpers excluded
go vet ./...
gofmt -w <files>     # format the files you change (no repo linter config)
```

CI (`.github/workflows/go.yml`): `go build -v ./...` then `go test -timeout 1m -covermode=atomic ./...`
on Linux. **Windows CI is build-only**, so keep OS-specific code behind build tags and don't break
the Windows build. There is no linter config: **green CI = build + vet + tests pass.**

Smoke-test the binary:

```bash
./plakar at /tmp/repo create
./plakar at /tmp/repo backup /some/dir
./plakar at /tmp/repo ls
```

## Where things live

| Path | What |
|---|---|
| `main.go`, `pkg.go` | CLI entry, arg dispatch |
| `subcommands/<cmd>/` | one dir per CLI command (`backup`, `restore`, `ls`, `check`, `sync`, `ptar`, `mount`, `ui`, ...) |
| `subcommands/subcommands.go` | `Subcommand` interface + `Register(...)` registry |
| `server/`, `api/` | HTTP server + API layer |
| `ui/` | embedded web UI wiring |
| `appcontext/` | shared `AppContext` threaded through commands |
| `services/`, `plugins/`, `reporting/`, `task/`, `login/`, `config/`, `cookies/` | supporting subsystems |
| `testing/`, `unittests/` | test fixtures/mocks (excluded from coverage) |
| `docs/`, `plakar.1`, `plakar-query.7` | docs and man pages |

**Add a CLI command:** create `subcommands/<name>/`, implement `Subcommand` (`Parse` + `Execute`,
embedding `SubcommandBase`), and `subcommands.Register(...)` in `init()`. Update `plakar.1` and
docs when user-facing behavior changes.

## Conventions

- **Tests:** table-driven `_test.go` alongside the code; keep each test under the 1-minute CI timeout.
- **Commit messages:** present tense, what + why (e.g. "Fix backup scheduler race").
- **Branches & PRs:** descriptive branch names (`fix-bug-issue123`, `feature-...`); PRs target `main`.
- **Cross-platform:** guard OS-specific code with build tags (see `term.go` / `term_windows.go`).
- **Dependencies:** add sparingly; avoid viral/GPL licenses; discuss non-trivial additions first.
- **Never commit** the `plakar` binary, `coverage.out`, or `junit.xml`, or hand-edit
  generated / man-page-synced content without regenerating it.
