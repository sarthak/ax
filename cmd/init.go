package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sarthak/ax/internal"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ax session",
	Long: `Initialize a new ax session.

If already inside tmux, configures the current session for ax (sets AX_SESSION_ID
and AX_SESSION_DIR environment variables) and creates the session directory.

If not inside tmux, creates a new tmux session named ax-<id>, configures it,
creates the session directory, then attaches to the new session.`,
	Example: `  ax init`,
	PreRunE: preRunNoExistingSession,
	RunE:    runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	sessionID, err := internal.GenerateSessionID()
	if err != nil {
		return fmt.Errorf("generate session id: %w", err)
	}

	paths, err := internal.InitSessionDir(sessionID)
	if err != nil {
		return fmt.Errorf("init session dir: %w", err)
	}

	tmux := internal.NewTmuxClient()

	if tmux.IsInsideTmux() {
		if err := tmux.SetEnv("AX_SESSION_ID", sessionID); err != nil {
			return fmt.Errorf("set AX_SESSION_ID: %w", err)
		}
		if err := tmux.SetEnv("AX_SESSION_DIR", paths.Root); err != nil {
			return fmt.Errorf("set AX_SESSION_DIR: %w", err)
		}
		cmd.Printf("ax session initialized\nsession: %s\nstate:   %s\n", sessionID, paths.Root)
		return nil
	}

	// Not inside tmux: create a detached session with env vars set at creation
	// time so the initial pane's shell inherits them immediately.
	sessionName := "ax-" + sessionID
	env := map[string]string{
		"AX_SESSION_ID":  sessionID,
		"AX_SESSION_DIR": paths.Root,
	}
	if err := tmux.NewSession(sessionName, env); err != nil {
		return fmt.Errorf("create tmux session: %w", err)
	}

	cmd.Printf("ax session initialized\nsession: %s\nstate:   %s\nattaching...\n", sessionID, paths.Root)

	// Replace current process with tmux attach-session so the terminal becomes
	// the new tmux session.
	return tmux.AttachSession(sessionName)
}
