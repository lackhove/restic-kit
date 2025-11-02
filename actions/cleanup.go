package actions

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CleanupConfig holds configuration for cleanup operations
type CleanupConfig struct {
	// No configuration needed for cleanup action
}

// ValidateCleanupConfig validates the cleanup config
func ValidateCleanupConfig(cfg *CleanupConfig) error {
	// No validation needed for cleanup config
	return nil
}

type CleanupAction struct {
	*BaseAction
	config *CleanupConfig
}

func NewCleanupAction(cfg *CleanupConfig) *CleanupAction {
	return &CleanupAction{
		BaseAction: NewBaseAction("cleanup"),
		config:     cfg,
	}
}

func (a *CleanupAction) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cleanup requires exactly one argument: the path to the log directory")
	}

	logDir := args[0]

	// Check if directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return fmt.Errorf("log directory does not exist: %s", logDir)
	}

	// Analyze backup results to determine overall success
	_, overallSuccess, err := analyzeBackupResults(logDir)
	if err != nil {
		return fmt.Errorf("failed to analyze backup results: %w", err)
	}

	if overallSuccess {
		// All backups successful, remove the directory
		if err := os.RemoveAll(logDir); err != nil {
			return fmt.Errorf("failed to remove log directory %s: %w", logDir, err)
		}
		fmt.Printf("Cleanup completed: removed log directory %s\n", logDir)
	} else {
		// Some backups failed, keep directory for debugging
		fmt.Printf("Cleanup skipped: keeping log directory %s for debugging (backup failures detected)\n", logDir)
	}

	return nil
}

func NewCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup [log-directory]",
		Short: "Clean up log directory after backup operations",
		Long:  `Remove the log directory if all backup operations were successful. Keep it for debugging if any operations failed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cleanupConfig := &CleanupConfig{}

			if err := ValidateCleanupConfig(cleanupConfig); err != nil {
				return fmt.Errorf("invalid cleanup config: %w", err)
			}

			action := NewCleanupAction(cleanupConfig)
			return action.Execute(args)
		},
	}

	return cmd
}
