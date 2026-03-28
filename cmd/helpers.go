package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sarthakagrawal/ax/internal"
)

// Package-level session state populated by preRunRequireSession.
var (
	currentSessionID string
	currentPaths     *internal.SessionPaths
)

// preRunRequireSession is a PreRunE hook for commands that need an active ax
// session (join, spawn, send, who, whoami). It validates the tmux environment
// and stores session state in package-level vars for use in RunE.
func preRunRequireSession(_ *cobra.Command, _ []string) error {
	tmux := internal.NewTmuxClient()

	if !tmux.IsInsideTmux() {
		return fmt.Errorf("not inside a tmux session — run `ax init` first")
	}

	sessionID, err := tmux.GetEnv("AX_SESSION_ID")
	if err != nil {
		return fmt.Errorf("AX_SESSION_ID not set — run `ax init` first")
	}

	paths, err := internal.LoadSessionPaths(sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	currentSessionID = sessionID
	currentPaths = paths
	return nil
}

// agentInfo is the shared JSON output structure for who and whoami.
type agentInfo struct {
	Label   string `json:"label"`
	Role    string `json:"role"`
	Session string `json:"session"`
	Pane    string `json:"pane"`
	Status  string `json:"status"`
}

// preRunNoExistingSession is a PreRunE hook for ax init. It ensures we are not
// already inside an ax session.
func preRunNoExistingSession(_ *cobra.Command, _ []string) error {
	tmux := internal.NewTmuxClient()
	if tmux.IsInsideTmux() {
		if _, err := tmux.GetEnv("AX_SESSION_ID"); err == nil {
			return fmt.Errorf("already inside an ax session (AX_SESSION_ID is set)")
		}
	}
	return nil
}
