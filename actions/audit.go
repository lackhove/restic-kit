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
	gomail "gopkg.in/gomail.v2"
)

// AuditConfig holds configuration for audit checks
type AuditConfig struct {
	GrowThreshold   float64
	ShrinkThreshold float64
	KeepHourly      int
	KeepDaily       int
	KeepWeekly      int
	KeepMonthly     int
	KeepYearly      int
	*NotifyEmailConfig
}

// ValidateAuditConfig validates the audit config
func ValidateAuditConfig(cfg *AuditConfig) error {
	if cfg.GrowThreshold < 0 {
		return fmt.Errorf("grow-threshold must be non-negative")
	}
	if cfg.ShrinkThreshold < 0 {
		return fmt.Errorf("shrink-threshold must be non-negative")
	}
	if cfg.NotifyEmailConfig != nil {
		return ValidateNotifyEmailConfig(cfg.NotifyEmailConfig)
	}
	return nil
}

// AuditCheckResult represents a failed audit check
type AuditCheckResult struct {
	CheckType string
	Path      string
	Message   string
	Details   map[string]string
}

// AuditAction performs audit checks on snapshots
type AuditAction struct {
	*BaseAction
	config *AuditConfig
}

func NewAuditAction(cfg *AuditConfig) *AuditAction {
	return &AuditAction{
		BaseAction: NewBaseAction("audit"),
		config:     cfg,
	}
}

func (a *AuditAction) Execute(args []string, dryRun bool) error {
	if len(args) != 1 {
		return fmt.Errorf("audit requires exactly one argument: the path to the log directory")
	}

	logDir := args[0]

	// Read snapshots from snapshots.out
	snapshots, err := a.readSnapshots(logDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshots: %w", err)
	}

	// Perform audit checks
	var failedChecks []AuditCheckResult

	// Check size changes
	sizeViolations := a.checkSizeChanges(snapshots)
	failedChecks = append(failedChecks, sizeViolations...)

	// Check retention policy
	retentionViolations := a.checkRetentionPolicy(snapshots)
	failedChecks = append(failedChecks, retentionViolations...)

	// Send email if there are failures and email config is provided
	if len(failedChecks) > 0 && a.config.NotifyEmailConfig != nil {
		if err := a.sendAuditEmail(failedChecks, dryRun); err != nil {
			return fmt.Errorf("failed to send audit email: %w", err)
		}
	}

	// Report results
	if len(failedChecks) > 0 {
		fmt.Printf("Audit FAILED: %d checks failed\n", len(failedChecks))
		for _, check := range failedChecks {
			fmt.Printf("- %s: %s\n", check.CheckType, check.Message)
		}
		return fmt.Errorf("audit checks failed")
	}

	fmt.Println("Audit PASSED: All checks successful")
	return nil
}

func (a *AuditAction) readSnapshots(logDir string) ([]Snapshot, error) {
	snapshotsFile := filepath.Join(logDir, "snapshots.out")
	content, err := os.ReadFile(snapshotsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots file: %w", err)
	}

	return parseSnapshotsOutput(string(content))
}

func (a *AuditAction) checkSizeChanges(snapshots []Snapshot) []AuditCheckResult {
	var violations []AuditCheckResult

	// Group snapshots by path
	groupedByPath := make(map[string][]Snapshot)
	for _, snap := range snapshots {
		key := strings.Join(snap.Paths, ", ")
		groupedByPath[key] = append(groupedByPath[key], snap)
	}

	for path, snaps := range groupedByPath {
		if len(snaps) < 2 {
			continue // Need at least 2 snapshots to compare
		}

		// Sort by time
		sort.Slice(snaps, func(i, j int) bool {
			t1, _ := time.Parse(time.RFC3339Nano, snaps[i].Time)
			t2, _ := time.Parse(time.RFC3339Nano, snaps[j].Time)
			return t1.Before(t2)
		})

		// Compare only the two most recent snapshots
		prev := snaps[len(snaps)-2] // Second most recent
		curr := snaps[len(snaps)-1] // Most recent

		if prev.Summary.TotalBytesProcessed == 0 {
			continue // Skip if previous size is 0
		}

		changePercent := float64(curr.Summary.TotalBytesProcessed-prev.Summary.TotalBytesProcessed) / float64(prev.Summary.TotalBytesProcessed) * 100

		var threshold float64
		var checkType string
		if changePercent > 0 {
			threshold = a.config.GrowThreshold
			checkType = "size_growth"
		} else {
			threshold = a.config.ShrinkThreshold
			checkType = "size_shrink"
			changePercent = -changePercent // Make positive for comparison
		}

		if changePercent >= threshold {
			violations = append(violations, AuditCheckResult{
				CheckType: checkType,
				Path:      path,
				Message:   fmt.Sprintf("%.1f%% change exceeds %.1f%% threshold", changePercent, threshold),
				Details: map[string]string{
					"previous_size":  formatBytes(prev.Summary.TotalBytesProcessed),
					"current_size":   formatBytes(curr.Summary.TotalBytesProcessed),
					"change_percent": fmt.Sprintf("%.1f", changePercent),
					"threshold":      fmt.Sprintf("%.1f", threshold),
					"previous_time":  prev.Time,
					"current_time":   curr.Time,
				},
			})
		}
	}

	return violations
}

func (a *AuditAction) checkRetentionPolicy(snapshots []Snapshot) []AuditCheckResult {
	var violations []AuditCheckResult

	// Group snapshots by path
	groupedByPath := make(map[string][]Snapshot)
	for _, snap := range snapshots {
		key := strings.Join(snap.Paths, ", ")
		groupedByPath[key] = append(groupedByPath[key], snap)
	}

	for path, snaps := range groupedByPath {
		// Parse times and sort
		type snapshotWithTime struct {
			snapshot Snapshot
			time     time.Time
		}

		var snapsWithTime []snapshotWithTime
		for _, snap := range snaps {
			t, err := time.Parse(time.RFC3339Nano, snap.Time)
			if err != nil {
				continue
			}
			snapsWithTime = append(snapsWithTime, snapshotWithTime{snapshot: snap, time: t})
		}

		sort.Slice(snapsWithTime, func(i, j int) bool {
			return snapsWithTime[i].time.Before(snapsWithTime[j].time)
		})

		// Check each retention policy
		policies := []struct {
			name    string
			keep    int
			groupBy func(time.Time) string
		}{
			{"hourly", a.config.KeepHourly, func(t time.Time) string {
				return fmt.Sprintf("%d-%02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour())
			}},
			{"daily", a.config.KeepDaily, func(t time.Time) string {
				return fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
			}},
			{"weekly", a.config.KeepWeekly, func(t time.Time) string {
				// Start week on Monday
				weekStart := t.AddDate(0, 0, -int(t.Weekday()-time.Monday+7)%7)
				return fmt.Sprintf("%d-%02d-%02d", weekStart.Year(), weekStart.Month(), weekStart.Day())
			}},
			{"monthly", a.config.KeepMonthly, func(t time.Time) string {
				return fmt.Sprintf("%d-%02d", t.Year(), t.Month())
			}},
			{"yearly", a.config.KeepYearly, func(t time.Time) string {
				return fmt.Sprintf("%d", t.Year())
			}},
		}

		for _, policy := range policies {
			if policy.keep == 0 {
				continue
			}

			// Group snapshots by time bucket
			buckets := make(map[string][]snapshotWithTime)
			for _, snap := range snapsWithTime {
				bucket := policy.groupBy(snap.time)
				buckets[bucket] = append(buckets[bucket], snap)
			}

			// Count buckets with snapshots
			bucketCount := len(buckets)

			if bucketCount > policy.keep {
				violations = append(violations, AuditCheckResult{
					CheckType: "retention_" + policy.name,
					Path:      path,
					Message:   fmt.Sprintf("too many %s snapshots: %d > %d", policy.name, bucketCount, policy.keep),
					Details: map[string]string{
						"policy":      policy.name,
						"actual":      strconv.Itoa(bucketCount),
						"expected":    strconv.Itoa(policy.keep),
						"total_snaps": strconv.Itoa(len(snapsWithTime)),
					},
				})
			}
		}
	}

	return violations
}

func (a *AuditAction) sendAuditEmail(failedChecks []AuditCheckResult, dryRun bool) error {
	subject := "Audit Report: FAILURES DETECTED"
	body := a.generateAuditEmailBody(failedChecks)

	if dryRun {
		fmt.Println("DRY RUN: Would send audit email with subject:", subject)
		fmt.Println("DRY RUN: Email body preview:")
		fmt.Println(body)
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", a.config.From)
	m.SetHeader("To", a.config.To)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(a.config.SMTPHost, a.config.SMTPPort, a.config.SMTPUsername, a.config.SMTPPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Audit email sent successfully")
	return nil
}

func (a *AuditAction) generateAuditEmailBody(failedChecks []AuditCheckResult) string {
	var body strings.Builder

	body.WriteString("Audit Report: FAILURES DETECTED\n\n")
	body.WriteString(fmt.Sprintf("Total failed checks: %d\n\n", len(failedChecks)))

	// Group by check type
	checksByType := make(map[string][]AuditCheckResult)
	for _, check := range failedChecks {
		checksByType[check.CheckType] = append(checksByType[check.CheckType], check)
	}

	for checkType, checks := range checksByType {
		body.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(checkType)))
		for _, check := range checks {
			body.WriteString(fmt.Sprintf("Path: %s\n", check.Path))
			body.WriteString(fmt.Sprintf("Issue: %s\n", check.Message))
			if len(check.Details) > 0 {
				body.WriteString("Details:\n")
				for key, value := range check.Details {
					body.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
				}
			}
			body.WriteString("\n")
		}
		body.WriteString("\n")
	}

	return body.String()
}

func NewAuditCmd() *cobra.Command {
	var growThreshold, shrinkThreshold float64
	var keepHourly, keepDaily, keepWeekly, keepMonthly, keepYearly int
	var smtpHost, smtpUsername, smtpPassword, from, to string
	var smtpPort int

	cmd := &cobra.Command{
		Use:   "audit [log-directory]",
		Short: "Audit snapshots for anomalies and retention policy compliance",
		Long: `Audit restic snapshots for size anomalies and retention policy compliance.
Checks for unusual size changes between snapshots and verifies snapshot counts
against retention policies. Sends email notifications for any failures.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var emailConfig *NotifyEmailConfig
			if smtpHost != "" || smtpUsername != "" || smtpPassword != "" || from != "" || to != "" {
				emailConfig = &NotifyEmailConfig{
					SMTPHost:     smtpHost,
					SMTPPort:     smtpPort,
					SMTPUsername: smtpUsername,
					SMTPPassword: smtpPassword,
					From:         from,
					To:           to,
				}
			}

			auditConfig := &AuditConfig{
				GrowThreshold:     growThreshold,
				ShrinkThreshold:   shrinkThreshold,
				KeepHourly:        keepHourly,
				KeepDaily:         keepDaily,
				KeepWeekly:        keepWeekly,
				KeepMonthly:       keepMonthly,
				KeepYearly:        keepYearly,
				NotifyEmailConfig: emailConfig,
			}

			if err := ValidateAuditConfig(auditConfig); err != nil {
				return fmt.Errorf("invalid audit config: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")

			action := NewAuditAction(auditConfig)
			return action.Execute(args, dryRun)
		},
	}

	cmd.Flags().Float64Var(&growThreshold, "grow-threshold", 20.0, "Maximum allowed growth percentage between snapshots")
	cmd.Flags().Float64Var(&shrinkThreshold, "shrink-threshold", 5.0, "Maximum allowed shrink percentage between snapshots")
	cmd.Flags().IntVar(&keepHourly, "keep-hourly", 0, "Number of hourly snapshots to keep")
	cmd.Flags().IntVar(&keepDaily, "keep-daily", 0, "Number of daily snapshots to keep")
	cmd.Flags().IntVar(&keepWeekly, "keep-weekly", 0, "Number of weekly snapshots to keep")
	cmd.Flags().IntVar(&keepMonthly, "keep-monthly", 0, "Number of monthly snapshots to keep")
	cmd.Flags().IntVar(&keepYearly, "keep-yearly", 0, "Number of yearly snapshots to keep")

	// Email flags (optional)
	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server hostname")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	cmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "SMTP username")
	cmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "SMTP password")
	cmd.Flags().StringVar(&from, "from", "", "From email address")
	cmd.Flags().StringVar(&to, "to", "", "To email address")

	return cmd
}
