package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sarthakagrawal/ax/internal"
)

var (
	whoRole string
	whoJSON bool
)

var whoCmd = &cobra.Command{
	Use:   "who",
	Short: "List agents registered in the current session",
	Long:  "List all agents registered in the current ax session, showing their label, role, pane ID, and live status.",
	Example: `  ax who
  ax who --role reviewer
  ax who --json`,
	PreRunE: preRunRequireSession,
	RunE:    runWho,
}

func init() {
	whoCmd.Flags().StringVar(&whoRole, "role", "", "Filter output to agents with this role")
	whoCmd.Flags().BoolVar(&whoJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(whoCmd)
}

func runWho(cmd *cobra.Command, _ []string) error {
	registry := internal.NewRegistry(currentPaths.Registry)

	var agents []internal.Agent
	if whoRole != "" {
		agents = registry.FindByRole(whoRole)
	} else {
		var err error
		agents, err = registry.Load()
		if err != nil {
			return fmt.Errorf("load registry: %w", err)
		}
	}

	tmux := internal.NewTmuxClient()

	entries := make([]agentInfo, len(agents))
	for i, agent := range agents {
		alive, err := tmux.PaneExists(agent.TmuxPaneID)
		if err != nil {
			return fmt.Errorf("check pane for %q: %w", agent.Label, err)
		}
		status := "dead"
		if alive {
			status = "alive"
		}
		entries[i] = agentInfo{
			Label:   agent.Label,
			Role:    agent.Role,
			Session: currentSessionID,
			Pane:    agent.TmuxPaneID,
			Status:  status,
		}
	}

	if whoJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "LABEL\tROLE\tPANE\tSTATUS")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Label, e.Role, e.Pane, e.Status)
	}
	return w.Flush()
}
