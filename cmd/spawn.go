package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/sarthak/ax/internal"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn --role <role> --label <label> [--layout hsplit|vsplit|window] \"<command>\"",
	Short: "Spawn a new agent in a tmux pane",
	Long: `Spawn creates a new tmux pane, registers the agent into the session registry,
and then starts the specified command in that pane.

The agent is registered BEFORE its process starts, so the spawning agent can
immediately send it messages without race conditions.

Supported layouts:
  hsplit   Horizontal split — new pane to the right (default)
  vsplit   Vertical split — new pane below
  window   New tmux window/tab

For layout requirements beyond what --layout provides (e.g., specific pane
sizes, moving a pane to a different window, swapping pane positions), use tmux
CLI commands directly after ax spawn completes. Pane IDs (%N) are stable across
rearrangement operations, so the session registry remains valid after any tmux
layout change.`,
	Example: `  ax spawn --role reviewer --label codex-1 "claude"
  ax spawn --role implementer --label claude-2 --layout vsplit "claude"

  # adjust layout with tmux directly after spawning:
  ax spawn --role reviewer --label codex-1 "claude"
  tmux resize-pane -t %5 -x 80`,
	Args:    cobra.ExactArgs(1),
	PreRunE: preRunRequireSession,
	RunE:    runSpawn,
}

var (
	spawnRole   string
	spawnLabel  string
	spawnLayout string
)

func init() {
	spawnCmd.Flags().StringVar(&spawnRole, "role", "", "Role of the spawned agent (required)")
	spawnCmd.Flags().StringVar(&spawnLabel, "label", "", "Unique label for the spawned agent (required)")
	spawnCmd.Flags().StringVar(&spawnLayout, "layout", "hsplit", "Pane layout: hsplit, vsplit, or window")
	_ = spawnCmd.MarkFlagRequired("role")
	_ = spawnCmd.MarkFlagRequired("label")
	rootCmd.AddCommand(spawnCmd)
}

func runSpawn(cmd *cobra.Command, args []string) error {
	command := args[0]

	switch spawnLayout {
	case "hsplit", "vsplit", "window":
	default:
		return fmt.Errorf("invalid --layout %q: must be hsplit, vsplit, or window", spawnLayout)
	}

	tmux := internal.NewTmuxClient()

	var paneID string
	var paneErr error
	switch spawnLayout {
	case "hsplit":
		paneID, paneErr = tmux.SplitH()
	case "vsplit":
		paneID, paneErr = tmux.SplitV()
	case "window":
		paneID, paneErr = tmux.NewWindow()
	}
	if paneErr != nil {
		return fmt.Errorf("create pane: %w", paneErr)
	}

	if err := tmux.CloseOnExit(paneID); err != nil {
		return fmt.Errorf("configure pane: %w", err)
	}

	commsDir := filepath.Join(currentPaths.Comms, spawnLabel)
	if err := os.MkdirAll(commsDir, 0755); err != nil {
		return fmt.Errorf("create comms dir: %w", err)
	}

	registry := internal.NewRegistry(currentPaths.Registry)
	agent := internal.Agent{
		Label:      spawnLabel,
		Role:       spawnRole,
		TmuxPaneID: paneID,
		Status:     internal.StatusAlive,
		CommsDir:   commsDir,
		JoinedAt:   time.Now(),
	}
	if err := registry.Upsert(agent); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	if err := tmux.SendKeys(paneID, command); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	cmd.Printf("Spawned agent %q (role: %s) in pane %s\n", spawnLabel, spawnRole, paneID)
	return nil
}
