package restic

import (
	"fmt"
)

// ResticMessage represents a message from restic JSON output
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

// SnapshotGroup represents a group of snapshots
type SnapshotGroup struct {
	GroupKey  GroupKey   `json:"group_key"`
	Snapshots []Snapshot `json:"snapshots"`
}

// GroupKey represents the key that groups snapshots
type GroupKey struct {
	Hostname string   `json:"hostname"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags"`
}

// Snapshot represents a restic snapshot
type Snapshot struct {
	Time           string        `json:"time"`
	Parent         string        `json:"parent"`
	Tree           string        `json:"tree"`
	Paths          []string      `json:"paths"`
	Hostname       string        `json:"hostname"`
	Username       string        `json:"username"`
	ProgramVersion string        `json:"program_version"`
	Summary        BackupSummary `json:"summary"`
	ID             string        `json:"id"`
	ShortID        string        `json:"short_id"`
}

// BackupSummary represents the summary of a backup operation
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

// ActionResult defines the interface for all action results
type ActionResult interface {
	GetActionName() string
	IsSuccess() bool
	GetSummaryInfo() map[string]string
	GetOutFile() string
	GetErrFile() string
}

// BackupResult represents the result of a backup operation
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

// BackupActionResult implements ActionResult for backup operations
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

// CheckResult represents the result of a check operation
type CheckResult struct {
	NumErrors int `json:"num_errors,omitempty"`
}

// CheckActionResult implements ActionResult for check operations
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

// SnapshotsActionResult implements ActionResult for snapshots operations
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

// ForgetActionResult implements ActionResult for forget operations
type ForgetActionResult struct {
	Name    string
	Success bool
	OutFile string
	ErrFile string
}

func (r *ForgetActionResult) GetActionName() string {
	return r.Name
}

func (r *ForgetActionResult) IsSuccess() bool {
	return r.Success
}

func (r *ForgetActionResult) GetSummaryInfo() map[string]string {
	info := make(map[string]string)
	status := "successful"
	if !r.Success {
		status = "failed"
	}
	info["status"] = status
	return info
}

func (r *ForgetActionResult) GetOutFile() string {
	return r.OutFile
}

func (r *ForgetActionResult) GetErrFile() string {
	return r.ErrFile
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
