package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "ax",
		Short: "ax - a simple CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
