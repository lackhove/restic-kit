package actions

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"restic-kit/shared"
)

func TestNotifyEmailAction(t *testing.T) {
	// Create a temporary directory with exitcode and output files
	tmpDir, err := os.MkdirTemp("", "logs*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create sample output files
	backupOut := `{"message_type":"summary","files_new":10,"files_changed":5,"files_unmodified":100,"dirs_new":2,"dirs_changed":1,"dirs_unmodified":50,"data_blobs":15,"tree_blobs":3,"data_added":1024000,"data_added_packed":512000,"total_files_processed":115,"total_bytes_processed":10485760,"total_duration":45.5,"snapshot_id":"abc123"}`
	checkOut := `{"message_type":"summary","num_errors":0}`
	snapshotsOut := `[{"group_key":{"hostname":"","paths":["/etc"],"tags":null},"snapshots":[{"time":"2023-10-01T10:00:00Z","summary":{"files_new":10,"files_changed":5,"files_unmodified":100,"dirs_new":2,"dirs_changed":1,"dirs_unmodified":50,"data_blobs":15,"tree_blobs":3,"data_added":1024000,"data_added_packed":512000,"total_files_processed":115,"total_bytes_processed":10485760},"id":"snap1"}]}]`

	// Create exitcode files (0 = success)
	os.WriteFile(tmpDir+"/backup.etc.exitcode", []byte("0"), 0644)
	os.WriteFile(tmpDir+"/check.exitcode", []byte("0"), 0644)
	os.WriteFile(tmpDir+"/snapshots.exitcode", []byte("0"), 0644)

	// Create output files
	os.WriteFile(tmpDir+"/backup.etc.out", []byte(backupOut), 0644)
	os.WriteFile(tmpDir+"/check.out", []byte(checkOut), 0644)
	os.WriteFile(tmpDir+"/snapshots.out", []byte(snapshotsOut), 0644)

	// Create email config (using a fake SMTP server for testing)
	emailConfig := &shared.NotifyEmailConfig{
		SMTPHost:     "localhost",
		SMTPPort:     2525, // Use a port that won't conflict
		SMTPUsername: "test",
		SMTPPassword: "test",
		From:         "from@example.com",
		To:           "to@example.com",
	}

	action := NewNotifyEmailAction(emailConfig)

	// Test with valid log directory
	err = action.Execute([]string{tmpDir}, false)
	// We expect this to fail because there's no SMTP server running, but it should parse logs successfully
	if err == nil {
		t.Log("Email action succeeded (unexpected in test environment)")
	} else {
		t.Logf("Email action failed as expected: %v", err)
	}

	// Test with missing directory
	err = action.Execute([]string{"nonexistent"}, false)
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}

	// Test with wrong number of arguments
	err = action.Execute([]string{}, false)
	if err == nil {
		t.Error("Expected error for no arguments, got nil")
	}

	err = action.Execute([]string{tmpDir, "extra"}, false)
	if err == nil {
		t.Error("Expected error for too many arguments, got nil")
	}
}

func TestNotifyEmailActionDryRun(t *testing.T) {
	// Create a temporary directory with exitcode and output files based on restic.log content
	tmpDir, err := os.MkdirTemp("", "logs-dry-run*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create sample output files based on actual restic log content
	backupDockerConfsOut := `{"message_type":"verbose_status","action":"unchanged","item":"/var/lib/docker-confs/.duplicacy","duration":0,"data_size":0,"data_size_in_repo":0,"metadata_size":0,"metadata_size_in_repo":0,"total_files":0}
{"message_type":"status","seconds_elapsed":1,"percent_done":0.000006725629195885543,"total_files":23,"files_done":1,"total_bytes":5650029,"bytes_done":38}
{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":298,"dirs_new":0,"dirs_changed":0,"dirs_unmodified":161,"data_blobs":0,"tree_blobs":0,"data_added":0,"data_added_packed":0,"total_files_processed":298,"total_bytes_processed":24547619,"total_duration":4.83816453,"backup_start":"2025-10-30T23:34:14.515777695+01:00","backup_end":"2025-10-30T23:34:19.35394226+01:00","snapshot_id":"466b2bc2f7c46c86a8d01e71853ee42e01c98f83549bfb42f4318ffccdbc85e5"}`

	nextcloudDataOut := `{"message_type":"verbose_status","action":"unchanged","item":"/var/mnt/datenhaufen/nextcloud_data/.duplicacy","duration":0,"data_size":0,"data_size_in_repo":0,"metadata_size":0,"metadata_size_in_repo":0,"total_files":0}
{"message_type":"status","seconds_elapsed":1,"percent_done":0.06990674346242294,"total_files":1013,"files_done":208,"total_bytes":3631366109,"bytes_done":253856979}
{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":298,"dirs_new":0,"dirs_changed":0,"dirs_unmodified":161,"data_blobs":0,"tree_blobs":0,"data_added":0,"data_added_packed":0,"total_files_processed":298,"total_bytes_processed":24547619,"total_duration":4.83816453,"backup_start":"2025-10-30T23:34:14.515777695+01:00","backup_end":"2025-10-30T23:34:19.35394226+01:00","snapshot_id":"466b2bc2f7c46c86a8d01e71853ee42e01c98f83549bfb42f4318ffccdbc85e5"}`

	snapshotsOut := `[{"group_key":{"hostname":"","paths":["/etc"],"tags":null},"snapshots":[{"time":"2025-10-30T23:34:19.35394226+01:00","summary":{"files_new":0,"files_changed":0,"files_unmodified":298,"dirs_new":0,"dirs_changed":0,"dirs_unmodified":161,"data_blobs":0,"tree_blobs":0,"data_added":0,"data_added_packed":0,"total_files_processed":298,"total_bytes_processed":24547619},"id":"466b2bc2f7c46c86a8d01e71853ee42e01c98f83549bfb42f4318ffccdbc85e5"}]}]`

	// Create exitcode files (0 = success)
	os.WriteFile(tmpDir+"/backup.docker-confs.exitcode", []byte("0"), 0644)
	os.WriteFile(tmpDir+"/backup.nextcloud_data.exitcode", []byte("0"), 0644)
	os.WriteFile(tmpDir+"/snapshots.exitcode", []byte("0"), 0644)

	// Create output files
	os.WriteFile(tmpDir+"/backup.docker-confs.out", []byte(backupDockerConfsOut), 0644)
	os.WriteFile(tmpDir+"/backup.nextcloud_data.out", []byte(nextcloudDataOut), 0644)
	os.WriteFile(tmpDir+"/snapshots.out", []byte(snapshotsOut), 0644)

	// Create email config
	emailConfig := &shared.NotifyEmailConfig{
		SMTPHost:     "localhost",
		SMTPPort:     2525,
		SMTPUsername: "test",
		SMTPPassword: "test",
		From:         "from@example.com",
		To:           "to@example.com",
	}

	action := NewNotifyEmailAction(emailConfig)

	// Capture stdout for validation
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test dry-run mode
	err = action.Execute([]string{tmpDir}, true)
	if err != nil {
		t.Errorf("Expected no error in dry-run mode, got %v", err)
	}

	// Restore stdout and capture output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Validate the output contains expected content
	expectedStrings := []string{
		"DRY RUN: Would send email with subject: Backup Report: SUCCESS",
		"DRY RUN: Email body preview:",
		"Overall Status: SUCCESS",
		"✅ backup docker-confs",
		"✅ backup nextcloud_data",
		"✅ snapshots",
		"Repository Snapshots: 1",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nActual output:\n%s", expected, output)
		}
	}
}

func TestValidateNotifyEmailConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *shared.NotifyEmailConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &shared.NotifyEmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     587,
				SMTPUsername: "user",
				SMTPPassword: "pass",
				From:         "from@example.com",
				To:           "to@example.com",
			},
			wantErr: false,
		},
		{
			name: "missing smtp-host",
			config: &shared.NotifyEmailConfig{
				SMTPUsername: "user",
				SMTPPassword: "pass",
				From:         "from@example.com",
				To:           "to@example.com",
			},
			wantErr: true,
			errMsg:  "smtp-host is required",
		},
		{
			name: "missing from",
			config: &shared.NotifyEmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				To:           "to@example.com",
			},
			wantErr: true,
			errMsg:  "from is required",
		},
		{
			name: "missing to",
			config: &shared.NotifyEmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPUsername: "user",
				SMTPPassword: "pass",
				From:         "from@example.com",
			},
			wantErr: true,
			errMsg:  "to is required",
		},
		{
			name: "missing smtp-username",
			config: &shared.NotifyEmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPPassword: "pass",
				From:         "from@example.com",
				To:           "to@example.com",
			},
			wantErr: true,
			errMsg:  "smtp-username is required",
		},
		{
			name: "missing smtp-password",
			config: &shared.NotifyEmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPUsername: "user",
				From:         "from@example.com",
				To:           "to@example.com",
			},
			wantErr: true,
			errMsg:  "smtp-password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateNotifyEmailConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate	shared.NotifyEmailConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Validate	shared.NotifyEmailConfig() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
