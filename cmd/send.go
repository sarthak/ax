package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sarthakagrawal/ax/internal"
)

var sendCmd = &cobra.Command{
	Use:   "send (<label> \"message\" | --role <role> \"message\")",
	Short: "Send a message to an agent or all agents with a role",
	Long: `Send delivers a short text message to a specific agent or broadcasts to all
agents with a given role.

Target by label:
  ax send <label> "message"

Broadcast by role (excludes self):
  ax send --role <role> "message"

Exactly one targeting mode must be provided. The sender must be registered
in the current session (run ` + "`ax join`" + ` first if not registered).

Messages are wrapped in the ax protocol format before delivery:
  [ax sms from <sender_label> (<sender_role>)]: <message>

If a target pane is no longer alive, its status is updated in the registry
and an error is reported. If all targets are dead, the command exits with an error.`,
	Example: `  ax send codex-1 "review ready, see comms/claude-1/plan.md"
  ax send --role reviewer "please review my latest changes"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if sendRole != "" {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return cobra.ExactArgs(2)(cmd, args)
	},
	PreRunE: preRunRequireSession,
	RunE:    runSend,
}

var sendRole string

func init() {
	sendCmd.Flags().StringVar(&sendRole, "role", "", "Broadcast to all agents with this role")
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	roleMode := sendRole != ""

	tmux := internal.NewTmuxClient()
	registry := internal.NewRegistry(currentPaths.Registry)

	// Identify the sender by current pane ID.
	currentPaneID, err := tmux.CurrentPaneID()
	if err != nil {
		return fmt.Errorf("get current pane: %w", err)
	}

	sender := registry.FindByPaneID(currentPaneID)
	if sender == nil {
		return fmt.Errorf("not registered as an agent in this session (run `ax join` first)")
	}

	// Resolve targets.
	var targets []internal.Agent
	var message string

	if roleMode {
		message = args[0]
		all := registry.FindByRole(sendRole)
		// Exclude self.
		for _, a := range all {
			if a.Label != sender.Label {
				targets = append(targets, a)
			}
		}
		if len(targets) == 0 {
			return fmt.Errorf("no agents with role %q found (excluding self)", sendRole)
		}
	} else {
		label := args[0]
		message = args[1]
		target := registry.FindByLabel(label)
		if target == nil {
			return fmt.Errorf("agent %q not found in registry", label)
		}
		targets = []internal.Agent{*target}
	}

	var sent, failed int
	var failedLabels []string

	for _, target := range targets {
		if !tmux.PaneExists(target.TmuxPaneID) {
			if updateErr := registry.UpdateStatus(target.Label, internal.StatusDead); updateErr != nil {
				// Log but don't mask the original dead-pane error.
				cmd.PrintErrf("warning: could not update status for %q: %v\n", target.Label, updateErr)
			}
			failed++
			failedLabels = append(failedLabels, target.Label)
			continue
		}
		if err := tmux.SendFormattedMessage(target.TmuxPaneID, sender.Label, sender.Role, message); err != nil {
			return fmt.Errorf("send to %s: %w", target.Label, err)
		}
		sent++
	}

	if sent > 0 {
		cmd.Printf("Sent to %d agent(s)\n", sent)
	}
	if failed > 0 {
		for _, label := range failedLabels {
			cmd.PrintErrf("error: pane for agent %q is dead (status updated to dead)\n", label)
		}
	}

	if sent == 0 {
		return fmt.Errorf("all targets failed (all panes are dead)")
	}
	return nil
}
