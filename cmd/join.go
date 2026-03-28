package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/sarthakagrawal/ax/internal"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Register the current pane into the ax session",
	Long: `Register the current tmux pane into the ax session.

Reads AX_SESSION_ID from the tmux environment, captures the current pane ID,
creates a comms directory for this agent, and upserts the agent into the
session registry. If the label already exists (e.g., after a restart), the
registry entry is updated with the current pane ID.`,
	Example: `  ax join --role implementer --label claude-1
  ax join --role reviewer --label codex-1`,
	PreRunE: preRunRequireSession,
	RunE:    runJoin,
}

var (
	joinRole  string
	joinLabel string
)

func init() {
	joinCmd.Flags().StringVar(&joinRole, "role", "", "agent role (e.g. implementer, reviewer)")
	joinCmd.Flags().StringVar(&joinLabel, "label", "", "unique agent label within the session (e.g. claude-1)")
	_ = joinCmd.MarkFlagRequired("role")
	_ = joinCmd.MarkFlagRequired("label")
	rootCmd.AddCommand(joinCmd)
}

func runJoin(cmd *cobra.Command, _ []string) error {
	tmux := internal.NewTmuxClient()

	paneID, err := tmux.CurrentPaneID()
	if err != nil {
		return fmt.Errorf("get current pane id: %w", err)
	}

	commsDir := filepath.Join(currentPaths.Comms, joinLabel)
	if err := os.MkdirAll(commsDir, 0755); err != nil {
		return fmt.Errorf("create comms dir: %w", err)
	}

	registry := internal.NewRegistry(currentPaths.Registry)

	// One pane can only be registered as one agent. If this pane is already
	// registered under a different label, reject the join.
	if existing := registry.FindByPaneID(paneID); existing != nil && existing.Label != joinLabel {
		return fmt.Errorf("pane %s is already registered as %q (use --label %s to update it)",
			paneID, existing.Label, existing.Label)
	}

	agent := internal.Agent{
		Label:      joinLabel,
		Role:       joinRole,
		TmuxPaneID: paneID,
		Status:     internal.StatusAlive,
		CommsDir:   commsDir,
		JoinedAt:   time.Now(),
	}
	if err := registry.Upsert(agent); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	cmd.Printf("joined ax session\nlabel:   %s\nrole:    %s\npane:    %s\nsession: %s\n",
		joinLabel, joinRole, paneID, currentSessionID)
	return nil
}
