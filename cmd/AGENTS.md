# cmd package

One file per CLI command. Commands are thin orchestrators — they call into `internal/` for all business logic.

## Command Registration Pattern

Each command is defined in its own file (e.g., `cmd/init.go`, `cmd/join.go`). The command variable and its `init()` function both live in that file:

```go
var myCmd = &cobra.Command{
    Use:     "my-command",
    Short:   "...",
    PreRunE: preRunRequireSession, // if session needed
    RunE:    runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

Flags are declared with `myCmd.Flags()` (local) inside the same `init()`. Required flags use `_ = myCmd.MarkFlagRequired("flag-name")`.

## PreRunE Hooks (`cmd/helpers.go`)

Precondition checks use Cobra's `PreRunE` to separate validation from business logic. Two hooks are available:

### `preRunRequireSession`

Validates that we're inside a tmux session with `AX_SESSION_ID` set and loads session paths. Sets package-level vars `currentSessionID` and `currentPaths` for use in `RunE`. Used by: `join`, `spawn`, `send`, `who`, `whoami`.

### `preRunNoExistingSession`

Validates that we're NOT already inside an ax session (i.e., `AX_SESSION_ID` is not set). Used by: `init`.

Do NOT use either hook for `skill` — it operates independently of sessions.

## Output Conventions

Use `cmd.Printf` and `cmd.PrintErrf` for all output — never `fmt.Printf` or `fmt.Fprintf(os.Stdout, ...)`. This routes output through Cobra's configurable writers, enabling testability via `cmd.SetOut()` and `cmd.SetErr()`. For tabwriter or other writer-wrapping, use `cmd.OutOrStdout()` as the underlying writer.

## Argument Validation

Use Cobra's built-in `Args` validators (`cobra.ExactArgs`, `cobra.MinimumNArgs`, `cobra.MaximumNArgs`, `cobra.RangeArgs`, `cobra.NoArgs`) instead of manual `len(args)` checks in `RunE`. For dynamic validation (e.g., arg count depends on a flag), use a custom `Args` function that delegates to the built-in validators:

```go
Args: func(cmd *cobra.Command, args []string) error {
    if someFlag != "" {
        return cobra.ExactArgs(1)(cmd, args)
    }
    return cobra.ExactArgs(2)(cmd, args)
},
```

## Flag Conventions

- Use `StringVar` / `BoolVar` with package-level variables for flag storage.
- Long flag names only (no single-character shorthand) to keep command signatures readable by agents.
- Mark mandatory flags with `MarkFlagRequired` immediately after declaration.

## Adding a New Command

1. Create `cmd/<name>.go`.
2. Declare `var <name>Cmd = &cobra.Command{...}` with `Use`, `Short`, `Long`, `Example`, `PreRunE`, and `RunE`.
3. In `init()`, declare flags and call `rootCmd.AddCommand(<name>Cmd)`.
4. Set `PreRunE` to the appropriate hook (`preRunRequireSession` or `preRunNoExistingSession`).
5. Use `cmd.Printf` / `cmd.PrintErrf` for output, and `currentSessionID` / `currentPaths` for session state.
6. Add a test file `cmd/<name>_test.go` verifying the command is registered and required flags are present.
7. Update this file if the command introduces a new pattern.

## When to update this file

Update when: adding or removing commands, changing flag conventions, modifying PreRunE hooks, or changing the command registration pattern.
