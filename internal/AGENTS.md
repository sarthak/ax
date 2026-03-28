# internal package

Shared logic for all `ax` commands. Three files, each owning a distinct concern.

## Package Boundaries

| File | Owns | Key Types |
|------|------|-----------|
| `session.go` | Filesystem paths, session ID generation, directory creation | `SessionPaths` |
| `tmux.go` | All subprocess calls to `tmux` binary | `TmuxClient` |
| `registry.go` | `registry.json` I/O, agent CRUD, querying | `Registry`, `Agent`, `AgentStatus` |

Do not mix these boundaries — e.g., `registry.go` must not shell out to tmux, and `tmux.go` must not read `registry.json`.

## Concurrency Contract

- **Registry writes** (`Upsert`, `UpdateStatus`) acquire an exclusive file lock (`syscall.Flock` on `registry.json.lock`) and use atomic writes (temp file + `os.Rename`).
- **Registry reads** (`Load`, `FindBy*`) are not locked. `Load()` caches after first disk read; mutations invalidate the cache.
- **`Save`** does not lock — it's only for initialization. Use `Upsert`/`UpdateStatus` for concurrent-safe mutations.

## Error Conventions

- Wrap errors with `fmt.Errorf("context: %w", err)` for traceability.
- Return errors to callers — never panic or `os.Exit` from this package. Exit handling belongs in `cmd/`.

## When to update this file

Update when: adding a new internal file, changing the caching/locking strategy, modifying package boundaries, or adding new exported types.
