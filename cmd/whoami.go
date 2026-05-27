package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sarthak/ax/internal"
)

var whoamiJSON bool

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current pane's registered agent identity",
	Long:  "Print the current pane's label, role, session ID, and pane ID as registered in the ax session.",
	Example: `  ax whoami
  ax whoami --json`,
	PreRunE: preRunRequireSession,
	RunE:    runWhoami,
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, _ []string) error {
	tmux := internal.NewTmuxClient()
	paneID, err := tmux.CurrentPaneID()
	if err != nil {
		return fmt.Errorf("get current pane: %w", err)
	}

	registry := internal.NewRegistry(currentPaths.Registry)
	agent := registry.FindByPaneID(paneID)
	if agent == nil {
		return fmt.Errorf("not registered as an agent in this session (run `ax join` first)")
	}

	if whoamiJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(agentInfo{
			Label:   agent.Label,
			Role:    agent.Role,
			Session: currentSessionID,
			Pane:    paneID,
			Status:  "alive",
		})
	}

	cmd.Printf("Label:      %s\n", agent.Label)
	cmd.Printf("Role:       %s\n", agent.Role)
	cmd.Printf("Session:    %s\n", currentSessionID)
	cmd.Printf("Pane:       %s\n", agent.TmuxPaneID)
	return nil
}
