package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"restic-kit/restic"
	"restic-kit/shared"
)

type NotifyEmailAction struct {
	*BaseAction
	config *shared.NotifyEmailConfig
}

func NewNotifyEmailAction(cfg *shared.NotifyEmailConfig) *NotifyEmailAction {
	return &NotifyEmailAction{
		BaseAction: NewBaseAction("notify-email"),
		config:     cfg,
	}
}

func (a *NotifyEmailAction) Execute(args []string, dryRun bool) error {
	if len(args) != 1 {
		return fmt.Errorf("notify-email requires exactly one argument: the path to the log directory")
	}

	logDir := args[0]

	actions, overallSuccess, err := analyzeBackupResults(logDir)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Backup Report: %s", map[bool]string{true: "SUCCESS", false: "FAILURE"}[overallSuccess])
	body := generateBodyFromActions(actions, overallSuccess)

	if dryRun {
		fmt.Println("DRY RUN: Would send email with subject:", subject)
		fmt.Println("DRY RUN: Email body preview:")
		fmt.Println(body)
		return nil
	}

	// Attach log files from action results
	var attachments []string
	for _, action := range actions {
		if action.IsSuccess() {
			continue
		}

		outFile := action.GetOutFile()
		errFile := action.GetErrFile()

		if outFile != "" {
			if _, err := os.Stat(outFile); err == nil {
				attachments = append(attachments, outFile)
			}
		}
		if errFile != "" {
			if _, err := os.Stat(errFile); err == nil {
				attachments = append(attachments, errFile)
			}
		}
	}

	if err := shared.SendEmail(a.config, subject, body, attachments, dryRun); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully")
	return nil
}

func generateBodyFromActions(actions []restic.ActionResult, success bool) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("Overall Status: %s\n\n", map[bool]string{true: "SUCCESS", false: "FAILURE"}[success]))

	// Process actions in execution order
	for _, action := range actions {
		switch actionResult := action.(type) {
		case *restic.BackupActionResult:
			statusEmoji := "✅"
			if !actionResult.Success {
				statusEmoji = "❌"
			}
			body.WriteString(fmt.Sprintf("%s backup %s\n", statusEmoji, actionResult.Name))

			info := actionResult.GetSummaryInfo()
			body.WriteString(fmt.Sprintf("  Files: %s new, %s changed, %s unmodified\n",
				info["files_new"], info["files_changed"], info["files_unmodified"]))
			body.WriteString(fmt.Sprintf("  Directories: %s new, %s changed, %s unmodified\n",
				info["dirs_new"], info["dirs_changed"], info["dirs_unmodified"]))
			body.WriteString(fmt.Sprintf("  Data added: %s (%s packed)\n",
				info["data_added"], info["data_added_packed"]))
			body.WriteString(fmt.Sprintf("  Total files processed: %s\n", info["total_files_processed"]))
			body.WriteString(fmt.Sprintf("  Total bytes processed: %s\n", info["total_bytes_processed"]))
			if duration, ok := info["duration"]; ok {
				body.WriteString(fmt.Sprintf("  Duration: %s seconds\n", duration))
			}
			body.WriteString("\n")

		case *restic.CheckActionResult:
			statusEmoji := "✅"
			if !actionResult.Success {
				statusEmoji = "❌"
			}
			body.WriteString(fmt.Sprintf("%s check\n", statusEmoji))
			info := actionResult.GetSummaryInfo()
			body.WriteString(fmt.Sprintf("  %s\n\n", info["status"]))

		case *restic.SnapshotsActionResult:
			body.WriteString(fmt.Sprintf("%s snapshots\n", "✅"))
			body.WriteString(fmt.Sprintf("  Repository Snapshots: %d\n", len(actionResult.Snapshots)))

			// Group snapshots by paths
			groupedByPath := make(map[string][]restic.Snapshot)
			for _, snap := range actionResult.Snapshots {
				key := strings.Join(snap.Paths, ", ")
				groupedByPath[key] = append(groupedByPath[key], snap)
			}

			// Sort paths alphabetically
			var paths []string
			for path := range groupedByPath {
				paths = append(paths, path)
			}
			sort.Strings(paths)

			for _, path := range paths {
				snapshots := groupedByPath[path]
				body.WriteString(fmt.Sprintf("\n  Path: %s\n", path))
				body.WriteString(fmt.Sprintf("  Snapshots: %d\n", len(snapshots)))

				if len(snapshots) > 0 {
					body.WriteString("  Date & Time          |      New | Modified |  Total Files |   Added Size |   Total Size\n")
					body.WriteString("  -------------------- | -------- | -------- | ------------ | ------------ | ------------\n")

					// Sort snapshots by time (newest first)
					sort.Slice(snapshots, func(i, j int) bool {
						return snapshots[i].Time > snapshots[j].Time
					})

					for _, snap := range snapshots {
						// Parse time for formatting (YYYY-MM-DD HH:MM)
						timeStr := snap.Time
						if len(timeStr) >= 16 {
							timeStr = timeStr[:10] + " " + timeStr[11:16] // YYYY-MM-DD HH:MM
						}

						// Get summary data
						newFiles := "0"
						modifiedFiles := "0"
						totalFiles := "0"
						addedSize := "0 B"
						totalSize := "0 B"

						if snap.Summary.FilesNew > 0 || snap.Summary.FilesChanged > 0 || snap.Summary.FilesUnmodified > 0 {
							newFiles = fmt.Sprintf("%d", snap.Summary.FilesNew)
							modifiedFiles = fmt.Sprintf("%d", snap.Summary.FilesChanged)
							totalFiles = fmt.Sprintf("%d", snap.Summary.TotalFilesProcessed)
							addedSize = formatBytes(snap.Summary.DataAdded)
							totalSize = formatBytes(snap.Summary.TotalBytesProcessed)
						}

						body.WriteString(fmt.Sprintf("  %-20s | %8s | %8s | %12s | %12s | %12s\n",
							timeStr, newFiles, modifiedFiles, totalFiles, addedSize, totalSize))
					}
				}
			}
			body.WriteString("\n")

		case *restic.ForgetActionResult:
			statusEmoji := "✅"
			if !actionResult.Success {
				statusEmoji = "❌"
			}
			body.WriteString(fmt.Sprintf("%s forget\n", statusEmoji))
			if actionResult.RemovedCount > 0 {
				body.WriteString(fmt.Sprintf("  %d snapshots removed\n\n", actionResult.RemovedCount))
			} else {
				body.WriteString("  no snapshots removed\n\n")
			}
		}
	}

	return body.String()
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// analyzeBackupResults analyzes the backup results from a log directory
// Helper functions (these could be moved to restic package if needed elsewhere)
func readExitCode(exitcodeFile string) (int, error) {
	content, err := os.ReadFile(exitcodeFile)
	if err != nil {
		return -1, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return -1, fmt.Errorf("invalid exit code in %s: %w", exitcodeFile, err)
	}
	return code, nil
}

func determineActionType(exitcodeFile string) (string, string) {
	base := filepath.Base(exitcodeFile)
	base = strings.TrimSuffix(base, ".exitcode")

	if strings.HasPrefix(base, "backup.") {
		actionName := strings.TrimPrefix(base, "backup.")
		return "backup", actionName
	} else if base == "check" {
		return "check", base
	} else if base == "snapshots" {
		return "snapshots", base
	} else if base == "forget" {
		return "forget", base
	}
	return "unknown", base
}

func determineOverallSuccessFromActions(actions []restic.ActionResult) bool {
	for _, action := range actions {
		if !action.IsSuccess() {
			return false
		}
	}
	return true
}

func analyzeBackupResults(logDir string) ([]restic.ActionResult, bool, error) {
	exitcodeFiles, err := filepath.Glob(filepath.Join(logDir, "*.exitcode"))
	if err != nil {
		return nil, false, fmt.Errorf("failed to list exitcode files in %s: %w", logDir, err)
	}

	// Sort exitcode files by modification time to preserve execution order
	type fileWithTime struct {
		path  string
		mtime time.Time
	}
	var filesWithTime []fileWithTime
	for _, f := range exitcodeFiles {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		filesWithTime = append(filesWithTime, fileWithTime{path: f, mtime: info.ModTime()})
	}
	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].mtime.Before(filesWithTime[j].mtime)
	})

	// Extract sorted file paths
	exitcodeFiles = make([]string, len(filesWithTime))
	for i, f := range filesWithTime {
		exitcodeFiles[i] = f.path
	}

	var actions []restic.ActionResult

	for _, exitcodeFile := range exitcodeFiles {
		actionType, actionName := determineActionType(exitcodeFile)

		exitCode, err := readExitCode(exitcodeFile)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read exit code from %s: %w", exitcodeFile, err)
		}

		success := exitCode == 0

		outFile := strings.TrimSuffix(exitcodeFile, ".exitcode") + ".out"
		errFile := strings.TrimSuffix(exitcodeFile, ".exitcode") + ".err"
		outContent, err := os.ReadFile(outFile)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read output file %s: %w", outFile, err)
		}

		switch actionType {
		case "backup":
			result, err := restic.ParseBackupOutput(string(outContent), success)
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse backup output for %s: %w", actionName, err)
			}
			actions = append(actions, &restic.BackupActionResult{
				Name:    actionName,
				Success: success,
				Result:  result,
				OutFile: outFile,
				ErrFile: errFile,
			})

		case "check":
			result, err := restic.ParseCheckOutput(string(outContent), success)
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse check output: %w", err)
			}
			actions = append(actions, &restic.CheckActionResult{
				Name:    actionName,
				Success: success,
				Result:  result,
				OutFile: outFile,
				ErrFile: errFile,
			})

		case "snapshots":
			snapshots, err := restic.ParseSnapshotsOutput(string(outContent))
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse snapshots output: %w", err)
			}
			actions = append(actions, &restic.SnapshotsActionResult{
				Name:      actionName,
				Success:   success,
				Snapshots: snapshots,
				OutFile:   outFile,
				ErrFile:   errFile,
			})

		case "forget":
			snapshots, removedCount, err := restic.ParseForgetOutput(string(outContent))
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse forget output: %w", err)
			}
			actions = append(actions, &restic.ForgetActionResult{
				Name:         actionName,
				Success:      success,
				Snapshots:    snapshots,
				RemovedCount: removedCount,
				OutFile:      outFile,
				ErrFile:      errFile,
			})
		}
	}

	overallSuccess := determineOverallSuccessFromActions(actions)
	return actions, overallSuccess, nil
}

func NewNotifyEmailCmd() *cobra.Command {
	var smtpHost, smtpUsername, smtpPassword, from, to string
	var smtpPort int

	cmd := &cobra.Command{
		Use:   "notify-email [log-directory]",
		Short: "Send an email notification",
		Long:  `Send an email notification using the configured SMTP settings. Parses JSON logs from the specified directory and generates a summary.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			emailConfig := &shared.NotifyEmailConfig{
				SMTPHost:     smtpHost,
				SMTPPort:     smtpPort,
				SMTPUsername: smtpUsername,
				SMTPPassword: smtpPassword,
				From:         from,
				To:           to,
			}

			if err := shared.ValidateNotifyEmailConfig(emailConfig); err != nil {
				return fmt.Errorf("invalid email config: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")

			action := NewNotifyEmailAction(emailConfig)
			return action.Execute(args, dryRun)
		},
	}

	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server hostname (required)")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	cmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "SMTP username (required)")
	cmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "SMTP password (required)")
	cmd.Flags().StringVar(&from, "from", "", "From email address (required)")
	cmd.Flags().StringVar(&to, "to", "", "To email address (required)")

	cmd.MarkFlagRequired("smtp-host")
	cmd.MarkFlagRequired("smtp-username")
	cmd.MarkFlagRequired("smtp-password")
	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")

	return cmd
}
