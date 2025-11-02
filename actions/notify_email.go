package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	gomail "gopkg.in/gomail.v2"
)

// NotifyEmailConfig holds configuration for email notifications
type NotifyEmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	From         string
	To           string
}

// ValidateNotifyEmailConfig validates the email notification config
func ValidateNotifyEmailConfig(cfg *NotifyEmailConfig) error {
	if cfg.SMTPHost == "" {
		return fmt.Errorf("smtp-host is required")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	if cfg.From == "" {
		return fmt.Errorf("from is required")
	}
	if cfg.To == "" {
		return fmt.Errorf("to is required")
	}
	if cfg.SMTPUsername == "" {
		return fmt.Errorf("smtp-username is required")
	}
	if cfg.SMTPPassword == "" {
		return fmt.Errorf("smtp-password is required")
	}
	return nil
}

type ResticMessage struct {
	MessageType string `json:"message_type"`
	// For summary
	FilesNew            int     `json:"files_new,omitempty"`
	FilesChanged        int     `json:"files_changed,omitempty"`
	FilesUnmodified     int     `json:"files_unmodified,omitempty"`
	DirsNew             int     `json:"dirs_new,omitempty"`
	DirsChanged         int     `json:"dirs_changed,omitempty"`
	DirsUnmodified      int     `json:"dirs_unmodified,omitempty"`
	DataBlobs           int     `json:"data_blobs,omitempty"`
	TreeBlobs           int     `json:"tree_blobs,omitempty"`
	DataAdded           int64   `json:"data_added,omitempty"`
	DataAddedPacked     int64   `json:"data_added_packed,omitempty"`
	TotalFilesProcessed int     `json:"total_files_processed,omitempty"`
	TotalBytesProcessed int64   `json:"total_bytes_processed,omitempty"`
	TotalDuration       float64 `json:"total_duration,omitempty"`
	SnapshotID          string  `json:"snapshot_id,omitempty"`
	// For check summary
	NumErrors int `json:"num_errors,omitempty"`
	// For status
	Message string `json:"message,omitempty"`
	// For snapshots
	Snapshots []SnapshotGroup `json:"snapshots,omitempty"`
}

type SnapshotGroup struct {
	GroupKey  GroupKey   `json:"group_key"`
	Snapshots []Snapshot `json:"snapshots"`
}

type GroupKey struct {
	Hostname string   `json:"hostname"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags"`
}

type Snapshot struct {
	Time    string        `json:"time"`
	Parent  string        `json:"parent"`
	Tree    string        `json:"tree"`
	Paths   []string      `json:"paths"`
	Summary BackupSummary `json:"summary"`
	ID      string        `json:"id"`
}

type BackupSummary struct {
	BackupStart         string `json:"backup_start"`
	BackupEnd           string `json:"backup_end"`
	FilesNew            int    `json:"files_new"`
	FilesChanged        int    `json:"files_changed"`
	FilesUnmodified     int    `json:"files_unmodified"`
	DirsNew             int    `json:"dirs_new"`
	DirsChanged         int    `json:"dirs_changed"`
	DirsUnmodified      int    `json:"dirs_unmodified"`
	DataBlobs           int    `json:"data_blobs"`
	TreeBlobs           int    `json:"tree_blobs"`
	DataAdded           int64  `json:"data_added"`
	DataAddedPacked     int64  `json:"data_added_packed"`
	TotalFilesProcessed int    `json:"total_files_processed"`
	TotalBytesProcessed int64  `json:"total_bytes_processed"`
}

type ActionResult interface {
	GetActionName() string
	IsSuccess() bool
	GetSummaryInfo() map[string]string
	GetOutFile() string
	GetErrFile() string
}

type BackupResult struct {
	FilesNew            int     `json:"files_new,omitempty"`
	FilesChanged        int     `json:"files_changed,omitempty"`
	FilesUnmodified     int     `json:"files_unmodified,omitempty"`
	DirsNew             int     `json:"dirs_new,omitempty"`
	DirsChanged         int     `json:"dirs_changed,omitempty"`
	DirsUnmodified      int     `json:"dirs_unmodified,omitempty"`
	DataAdded           int64   `json:"data_added,omitempty"`
	DataAddedPacked     int64   `json:"data_added_packed,omitempty"`
	TotalFilesProcessed int     `json:"total_files_processed,omitempty"`
	TotalBytesProcessed int64   `json:"total_bytes_processed,omitempty"`
	TotalDuration       float64 `json:"total_duration,omitempty"`
}

type BackupActionResult struct {
	Name    string
	Success bool
	Result  *BackupResult
	OutFile string
	ErrFile string
}

func (r *BackupActionResult) GetActionName() string {
	return r.Name
}

func (r *BackupActionResult) IsSuccess() bool {
	return r.Success
}

func (r *BackupActionResult) GetSummaryInfo() map[string]string {
	info := make(map[string]string)
	if r.Result != nil {
		info["files_new"] = fmt.Sprintf("%d", r.Result.FilesNew)
		info["files_changed"] = fmt.Sprintf("%d", r.Result.FilesChanged)
		info["files_unmodified"] = fmt.Sprintf("%d", r.Result.FilesUnmodified)
		info["dirs_new"] = fmt.Sprintf("%d", r.Result.DirsNew)
		info["dirs_changed"] = fmt.Sprintf("%d", r.Result.DirsChanged)
		info["dirs_unmodified"] = fmt.Sprintf("%d", r.Result.DirsUnmodified)
		info["data_added"] = formatBytes(r.Result.DataAdded)
		info["data_added_packed"] = formatBytes(r.Result.DataAddedPacked)
		info["total_files_processed"] = fmt.Sprintf("%d", r.Result.TotalFilesProcessed)
		info["total_bytes_processed"] = formatBytes(r.Result.TotalBytesProcessed)
		if r.Result.TotalDuration > 0 {
			info["duration"] = fmt.Sprintf("%.2f", r.Result.TotalDuration)
		}
	}
	return info
}

func (r *BackupActionResult) GetOutFile() string {
	return r.OutFile
}

func (r *BackupActionResult) GetErrFile() string {
	return r.ErrFile
}

type CheckResult struct {
	NumErrors int `json:"num_errors,omitempty"`
}

type CheckActionResult struct {
	Name    string
	Success bool
	Result  *CheckResult
	OutFile string
	ErrFile string
}

func (r *CheckActionResult) GetActionName() string {
	return r.Name
}

func (r *CheckActionResult) IsSuccess() bool {
	return r.Success
}

func (r *CheckActionResult) GetSummaryInfo() map[string]string {
	info := make(map[string]string)
	if r.Result != nil {
		info["num_errors"] = fmt.Sprintf("%d", r.Result.NumErrors)
		if r.Result.NumErrors == 0 {
			info["status"] = "PASSED"
		} else {
			info["status"] = "FAILED"
		}
	}
	return info
}

func (r *CheckActionResult) GetOutFile() string {
	return r.OutFile
}

func (r *CheckActionResult) GetErrFile() string {
	return r.ErrFile
}

type SnapshotsActionResult struct {
	Name      string
	Success   bool
	Snapshots []Snapshot
	OutFile   string
	ErrFile   string
}

func (r *SnapshotsActionResult) GetActionName() string {
	return r.Name
}

func (r *SnapshotsActionResult) IsSuccess() bool {
	return r.Success
}

func (r *SnapshotsActionResult) GetSummaryInfo() map[string]string {
	info := make(map[string]string)
	info["total_snapshots"] = fmt.Sprintf("%d", len(r.Snapshots))
	return info
}

func (r *SnapshotsActionResult) GetOutFile() string {
	return r.OutFile
}

func (r *SnapshotsActionResult) GetErrFile() string {
	return r.ErrFile
}

type NotifyEmailAction struct {
	*BaseAction
	config *NotifyEmailConfig
}

func NewNotifyEmailAction(cfg *NotifyEmailConfig) *NotifyEmailAction {
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

	m := gomail.NewMessage()
	m.SetHeader("From", a.config.From)
	m.SetHeader("To", a.config.To)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// Attach log files from action results
	for _, action := range actions {
		if action.IsSuccess() {
			continue
		}

		outFile := action.GetOutFile()
		errFile := action.GetErrFile()

		if outFile != "" {
			if _, err := os.Stat(outFile); err == nil {
				m.Attach(outFile)
			}
		}
		if errFile != "" {
			if _, err := os.Stat(errFile); err == nil {
				m.Attach(errFile)
			}
		}
	}

	d := gomail.NewDialer(a.config.SMTPHost, a.config.SMTPPort, a.config.SMTPUsername, a.config.SMTPPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully")
	return nil
}

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
	}
	return "unknown", base
}

func parseBackupOutput(content string, success bool) (*BackupResult, error) {
	lines := strings.Split(content, "\n")
	var lastLine string

	// Find the last non-empty line (summary is always on the last line)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lastLine = line
			break
		}
	}

	if lastLine == "" {
		return &BackupResult{}, nil
	}

	var msg ResticMessage
	if err := json.Unmarshal([]byte(lastLine), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse backup summary JSON: %w", err)
	}

	result := &BackupResult{
		FilesNew:            msg.FilesNew,
		FilesChanged:        msg.FilesChanged,
		FilesUnmodified:     msg.FilesUnmodified,
		DirsNew:             msg.DirsNew,
		DirsChanged:         msg.DirsChanged,
		DirsUnmodified:      msg.DirsUnmodified,
		DataAdded:           msg.DataAdded,
		DataAddedPacked:     msg.DataAddedPacked,
		TotalFilesProcessed: msg.TotalFilesProcessed,
		TotalBytesProcessed: msg.TotalBytesProcessed,
		TotalDuration:       msg.TotalDuration,
	}

	return result, nil
}

func parseCheckOutput(content string, success bool) (*CheckResult, error) {
	var msg ResticMessage
	if err := json.Unmarshal([]byte(content), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse check output as JSON: %w", err)
	}

	result := &CheckResult{
		NumErrors: msg.NumErrors,
	}
	return result, nil
}

func parseSnapshotsOutput(content string) ([]Snapshot, error) {
	var snapshotGroups []SnapshotGroup
	if err := json.Unmarshal([]byte(content), &snapshotGroups); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots output as JSON: %w", err)
	}

	var snapshots []Snapshot
	for _, group := range snapshotGroups {
		snapshots = append(snapshots, group.Snapshots...)
	}
	return snapshots, nil
}

func determineOverallSuccessFromActions(actions []ActionResult) bool {
	for _, action := range actions {
		if !action.IsSuccess() {
			return false
		}
	}
	return true
}

func generateBodyFromActions(actions []ActionResult, success bool) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("Overall Status: %s\n\n", map[bool]string{true: "SUCCESS", false: "FAILURE"}[success]))

	var backupActions []ActionResult
	var checkActions []ActionResult
	var allSnapshots []Snapshot

	for _, action := range actions {
		switch actionResult := action.(type) {
		case *BackupActionResult:
			backupActions = append(backupActions, action)
		case *CheckActionResult:
			checkActions = append(checkActions, action)
		case *SnapshotsActionResult:
			allSnapshots = append(allSnapshots, actionResult.Snapshots...)
		}
	}

	if len(backupActions) > 0 {
		body.WriteString("Backup Summaries:\n")
		for _, action := range backupActions {
			result := action.(*BackupActionResult)
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
			result := action.(*CheckActionResult)
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

		groupedByPath := make(map[string][]Snapshot)
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
					formatBytes(snap.Summary.DataAdded),
					formatBytes(snap.Summary.TotalBytesProcessed)))
			}
			body.WriteString("\n")
		}
	}

	return body.String()
}

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
// and returns the actions, overall success status, and any error
func analyzeBackupResults(logDir string) ([]ActionResult, bool, error) {
	exitcodeFiles, err := filepath.Glob(filepath.Join(logDir, "*.exitcode"))
	if err != nil {
		return nil, false, fmt.Errorf("failed to list exitcode files in %s: %w", logDir, err)
	}

	var actions []ActionResult

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
			result, err := parseBackupOutput(string(outContent), success)
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse backup output for %s: %w", actionName, err)
			}
			actions = append(actions, &BackupActionResult{
				Name:    actionName,
				Success: success,
				Result:  result,
				OutFile: outFile,
				ErrFile: errFile,
			})

		case "check":
			result, err := parseCheckOutput(string(outContent), success)
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse check output: %w", err)
			}
			actions = append(actions, &CheckActionResult{
				Name:    actionName,
				Success: success,
				Result:  result,
				OutFile: outFile,
				ErrFile: errFile,
			})

		case "snapshots":
			snapshots, err := parseSnapshotsOutput(string(outContent))
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse snapshots output: %w", err)
			}
			actions = append(actions, &SnapshotsActionResult{
				Name:      actionName,
				Success:   success,
				Snapshots: snapshots,
				OutFile:   outFile,
				ErrFile:   errFile,
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
			emailConfig := &NotifyEmailConfig{
				SMTPHost:     smtpHost,
				SMTPPort:     smtpPort,
				SMTPUsername: smtpUsername,
				SMTPPassword: smtpPassword,
				From:         from,
				To:           to,
			}

			if err := ValidateNotifyEmailConfig(emailConfig); err != nil {
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
