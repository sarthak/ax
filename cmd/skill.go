package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed skill.md
var skillContent string

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Print the ax skill file to stdout",
	Long:  "Print the ax SKILL.md to stdout. Install it into your coding agent's skill directory:\n\n  ax skill > ~/.claude/skills/ax.md",
	Example: `  ax skill
  ax skill > ~/.claude/skills/ax.md`,
	Args: cobra.NoArgs,
	RunE: runSkill,
}

func init() {
	rootCmd.AddCommand(skillCmd)
}

func runSkill(cmd *cobra.Command, _ []string) error {
	cmd.Print(skillContent)
	return nil
}
