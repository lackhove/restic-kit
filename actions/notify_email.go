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

	var backupActions []restic.ActionResult
	var checkActions []restic.ActionResult
	var forgetActions []restic.ActionResult
	var allSnapshots []restic.Snapshot

	for _, action := range actions {
		switch actionResult := action.(type) {
		case *restic.BackupActionResult:
			backupActions = append(backupActions, action)
		case *restic.CheckActionResult:
			checkActions = append(checkActions, action)
		case *restic.SnapshotsActionResult:
			allSnapshots = append(allSnapshots, actionResult.Snapshots...)
		case *restic.ForgetActionResult:
			forgetActions = append(forgetActions, action)
		}
	}

	if len(backupActions) > 0 {
		body.WriteString("Backup Summaries:\n")
		for _, action := range backupActions {
			result := action.(*restic.BackupActionResult)
			statusEmoji := "✅"
			if !result.Success {
				statusEmoji = "❌"
			}
			body.WriteString(fmt.Sprintf("\n%s %s:\n", statusEmoji, result.Name))

			info := result.GetSummaryInfo()
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
		}
	}

	if len(checkActions) > 0 {
		body.WriteString("\nRepository Check:\n")
		for _, action := range checkActions {
			result := action.(*restic.CheckActionResult)
			statusEmoji := "✅"
			if !result.Success {
				statusEmoji = "❌"
			}
			info := result.GetSummaryInfo()
			body.WriteString(fmt.Sprintf("%s %s\n", statusEmoji, info["status"]))
		}
	}

	if len(allSnapshots) > 0 {
		body.WriteString("\n\nRepository Snapshots:\n")
		body.WriteString(fmt.Sprintf("Total snapshots: %d\n\n", len(allSnapshots)))

		groupedByPath := make(map[string][]restic.Snapshot)
		for _, snap := range allSnapshots {
			key := strings.Join(snap.Paths, ", ")
			groupedByPath[key] = append(groupedByPath[key], snap)
		}

		var paths []string
		for path := range groupedByPath {
			paths = append(paths, path)
		}
		sort.Strings(paths)

		for _, path := range paths {
			snaps := groupedByPath[path]
			body.WriteString(fmt.Sprintf("Path: %s\n", path))
			body.WriteString(fmt.Sprintf("Snapshots: %d\n", len(snaps)))
			body.WriteString(fmt.Sprintf("%-20s | %8s | %8s | %12s | %12s | %12s\n", "Date & Time", "New", "Modified", "Total Files", "Added Size", "Total Size"))
			body.WriteString(fmt.Sprintf("%-20s | %8s | %8s | %12s | %12s | %12s\n", strings.Repeat("-", 20), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 12), strings.Repeat("-", 12), strings.Repeat("-", 12)))
			for _, snap := range snaps {
				parsedTime, err := time.Parse(time.RFC3339Nano, snap.Time)
				var timeStr string
				if err == nil {
					timeStr = parsedTime.Format("2006-01-02 15:04")
				} else {
					timeStr = snap.Time
					if len(timeStr) > 20 {
						timeStr = timeStr[:20]
					}
				}
				body.WriteString(fmt.Sprintf("%-20s | %8d | %8d | %12d | %12s | %12s\n",
					timeStr,
					snap.Summary.FilesNew,
					snap.Summary.FilesChanged,
					snap.Summary.TotalFilesProcessed,
					shared.FormatBytes(snap.Summary.DataAdded),
					shared.FormatBytes(snap.Summary.TotalBytesProcessed)))
			}
			body.WriteString("\n")
		}
	}

	if len(forgetActions) > 0 {
		body.WriteString("\n\nForget Operations:\n")
		for _, action := range forgetActions {
			result := action.(*restic.ForgetActionResult)
			statusEmoji := "✅"
			if !result.Success {
				statusEmoji = "❌"
			}
			statusText := "successful"
			if !result.Success {
				statusText = "failed"
			}
			body.WriteString(fmt.Sprintf("%s %s: %s\n",
				statusEmoji, result.Name, statusText))
		}
	}

	return body.String()
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
			actions = append(actions, &restic.ForgetActionResult{
				Name:    actionName,
				Success: success,
				OutFile: outFile,
				ErrFile: errFile,
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
