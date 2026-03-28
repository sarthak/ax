# ax — Agent Exchange CLI

A lightweight CLI enabling coding agents in tmux panes to discover each other and exchange short messages.

**Spec:** `docs/specs/001-ax.md`

## Build & Run

```bash
make build        # build the binary
make run          # go run .
make test         # unit tests (no tmux needed)
make fmt          # auto-fix formatting (goimports) and tidy go.mod
make lint         # run all linters and verify go.mod is tidy
make clean        # remove binary
```

**Always run `make fmt lint` after completing work** to ensure code is properly formatted and lint-clean.

## Project Structure

| Path | Purpose |
|------|---------|
| `main.go` | Entry point, delegates to `cmd.Execute()` |
| `cmd/` | One file per CLI command (`init`, `join`, `spawn`, `send`, `who`, `whoami`, `skill`) |
| `internal/` | Shared logic: `session.go`, `tmux.go`, `registry.go` (see `internal/AGENTS.md`) |
| `docs/specs/` | Product spec |

## Key Conventions

- **All tmux interaction** goes through `internal.TmuxClient`. Never shell out to tmux directly from `cmd/`.
- **All registry mutations** go through `Registry.Upsert` or `Registry.UpdateStatus`. Both use file locking and atomic writes.
- **CLAUDE.md pattern**: Every `CLAUDE.md` in this project is a single line `@AGENTS.md`. All content lives in the corresponding `AGENTS.md`. Follow this pattern when creating new ones.

## Testing

Use `github.com/stretchr/testify` for assertions:
- `require` for preconditions that should halt the test on failure (setup, essential checks)
- `assert` for the actual assertions being tested

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/stretchr/testify` — test assertions (assert/require)

## When to update this file

Update when: adding/removing commands, changing project structure, adding dependencies, or modifying conventions described above.
