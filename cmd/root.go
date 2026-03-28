package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ax",
	Short: "ax — Agent Exchange CLI",
	Long:  "ax enables coding agents running in tmux panes to discover each other and exchange short messages.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
