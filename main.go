package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"restic-kit/actions"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "restic-kit",
		Short: "Restic hooks for backup automation",
		Long:  `A tool for executing hooks during restic backup operations.`,
	}

	rootCmd.PersistentFlags().Bool("dry-run", false, "dry run mode")

	// Add action commands
	rootCmd.AddCommand(actions.NewNotifyEmailCmd())
	rootCmd.AddCommand(actions.NewNotifyHTTPCmd())
	rootCmd.AddCommand(actions.NewWaitOnlineCmd())
	rootCmd.AddCommand(actions.NewCleanupCmd())
	rootCmd.AddCommand(actions.NewAuditCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
