package restic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseBackupOutput parses backup JSON output
func ParseBackupOutput(content string, success bool) (*BackupResult, error) {
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

// ParseCheckOutput parses check JSON output
func ParseCheckOutput(content string, success bool) (*CheckResult, error) {
	var msg ResticMessage
	if err := json.Unmarshal([]byte(content), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse check output as JSON: %w", err)
	}

	result := &CheckResult{
		NumErrors: msg.NumErrors,
	}
	return result, nil
}

// ParseSnapshotsOutput parses snapshots JSON output
func ParseSnapshotsOutput(content string) ([]Snapshot, error) {
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

// readExitCode reads exit code from file
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

// determineActionType determines action type from exitcode filename
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

// determineOverallSuccess determines overall success from action results
func determineOverallSuccessFromActions(actions []ActionResult) bool {
	for _, action := range actions {
		if !action.IsSuccess() {
			return false
		}
	}
	return true
}
