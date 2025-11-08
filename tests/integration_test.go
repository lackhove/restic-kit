package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLINotifyHTTP(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "cli-http-test*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create successful exitcode files
	os.WriteFile(filepath.Join(tmpDir, "backup.test.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "forget.exitcode"), []byte("0"), 0644)

	// Create corresponding .out files
	os.WriteFile(filepath.Join(tmpDir, "backup.test.out"), []byte(`{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":100}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.out"), []byte(`{"message_type":"summary","num_errors":0}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.out"), []byte(`[]`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "forget.out"), []byte(`[{"tags":null,"host":"","paths":["/test"],"keep":[],"remove":[]}]`), 0644)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Build the binary
	binaryPath := filepath.Join(os.TempDir(), "restic-kit-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd")
	cmd.Dir = ".." // Go back to project root
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Run the CLI command with URL flag and log directory
	cmd = exec.Command(binaryPath, "notify-http", "--url", server.URL, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI command failed: %v, output: %s", err, string(output))
	}

	// Check that the output contains success message
	if string(output) == "" {
		t.Error("Expected output from CLI command")
	}
}

func TestCLIWaitOnline(t *testing.T) {
	// Create a test server that succeeds immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Build the binary
	binaryPath := filepath.Join(os.TempDir(), "restic-kit-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd")
	cmd.Dir = ".." // Go back to project root
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Run the CLI command with flags
	start := time.Now()
	cmd = exec.Command(binaryPath, "wait-online",
		"--url", server.URL,
		"--timeout", "1s",
		"--initial-delay", "10ms",
		"--max-delay", "100ms")
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CLI command failed: %v, output: %s", err, string(output))
	}

	// Should complete quickly since server responds immediately
	if duration > 100*time.Millisecond {
		t.Errorf("Expected quick completion, took %v", duration)
	}
}

func TestCLIAudit(t *testing.T) {
	// Use the existing test data directory
	logDir := "tests/restic-logs"

	// Build the binary
	binaryPath := filepath.Join(os.TempDir(), "restic-kit-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd")
	cmd.Dir = ".." // Go back to project root
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Test audit with default thresholds (should fail due to size changes)
	cmd = exec.Command(binaryPath, "audit", "--dry-run", logDir)
	cmd.Dir = ".." // Go back to project root
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected audit to fail with default thresholds, but it passed")
	}

	// Check that output contains failure information
	outputStr := string(output)
	if !contains(outputStr, "Audit FAILED") {
		t.Errorf("Expected 'Audit FAILED' in output, got: %s", outputStr)
	}
	if !contains(outputStr, "size_shrink") {
		t.Errorf("Expected size_shrink violations in output, got: %s", outputStr)
	}

	// Test audit with higher thresholds (should pass)
	cmd = exec.Command(binaryPath, "audit", "--dry-run", "--shrink-threshold", "20", "--grow-threshold", "50", logDir)
	cmd.Dir = ".." // Go back to project root
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Expected audit to pass with higher thresholds, but failed: %v, output: %s", err, outputStr)
	}

	// Check that output contains success message
	outputStr = string(output)
	if !contains(outputStr, "Audit PASSED") {
		t.Errorf("Expected 'Audit PASSED' in output, got: %s", outputStr)
	}
}

func TestCLINotifyEmail(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "cli-email-test*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files in chronological order with specific timestamps
	baseTime := time.Now().Add(-4 * time.Hour)

	// Create backup file first (oldest)
	backupTime := baseTime
	os.WriteFile(filepath.Join(tmpDir, "backup.test.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "backup.test.out"), []byte(`{"message_type":"summary","files_new":5,"files_changed":2,"files_unmodified":100,"dirs_new":1,"dirs_changed":0,"dirs_unmodified":50,"data_added":1024,"data_added_packed":512,"total_files_processed":107,"total_bytes_processed":1048576,"total_duration":10.5}`), 0644)
	os.Chtimes(filepath.Join(tmpDir, "backup.test.exitcode"), backupTime, backupTime)
	os.Chtimes(filepath.Join(tmpDir, "backup.test.out"), backupTime, backupTime)

	// Create check file second
	checkTime := baseTime.Add(1 * time.Hour)
	os.WriteFile(filepath.Join(tmpDir, "check.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.out"), []byte(`{"message_type":"summary","num_errors":0}`), 0644)
	os.Chtimes(filepath.Join(tmpDir, "check.exitcode"), checkTime, checkTime)
	os.Chtimes(filepath.Join(tmpDir, "check.out"), checkTime, checkTime)

	// Create snapshots file third
	snapshotsTime := baseTime.Add(2 * time.Hour)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.out"), []byte(`[{"group_key":{"hostname":"test","paths":["/test"],"tags":[]},"snapshots":[{"id":"abc123","time":"2025-11-01T10:00:00","paths":["/test"],"hostname":"test","username":"test","tags":[]}]}]`), 0644)
	os.Chtimes(filepath.Join(tmpDir, "snapshots.exitcode"), snapshotsTime, snapshotsTime)
	os.Chtimes(filepath.Join(tmpDir, "snapshots.out"), snapshotsTime, snapshotsTime)

	// Create forget file last (newest)
	forgetTime := baseTime.Add(3 * time.Hour)
	os.WriteFile(filepath.Join(tmpDir, "forget.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "forget.out"), []byte(`[{"tags":null,"host":"","paths":["/test"],"keep":[{"id":"abc123"}],"remove":[{"id":"def456"}]}]`), 0644)
	os.Chtimes(filepath.Join(tmpDir, "forget.exitcode"), forgetTime, forgetTime)
	os.Chtimes(filepath.Join(tmpDir, "forget.out"), forgetTime, forgetTime)

	// Build the binary
	binaryPath := filepath.Join(os.TempDir(), "restic-kit-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd")
	cmd.Dir = ".." // Go back to project root
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Run the CLI command in dry-run mode
	cmd = exec.Command(binaryPath, "notify-email",
		"--dry-run",
		"--smtp-host", "localhost",
		"--smtp-username", "test",
		"--smtp-password", "test",
		"--from", "test@example.com",
		"--to", "test@example.com",
		tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI command failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify that the output contains the expected sections in chronological order
	lines := strings.Split(outputStr, "\n")

	// Find the positions of each action in the output
	backupPos := -1
	checkPos := -1
	snapshotsPos := -1
	forgetPos := -1

	for i, line := range lines {
		if strings.Contains(line, "✅ backup test") {
			backupPos = i
		} else if strings.Contains(line, "✅ check") {
			checkPos = i
		} else if strings.Contains(line, "✅ snapshots") {
			snapshotsPos = i
		} else if strings.Contains(line, "✅ forget") {
			forgetPos = i
		}
	}

	// Verify chronological order: backup -> check -> snapshots -> forget
	if backupPos == -1 || checkPos == -1 || snapshotsPos == -1 || forgetPos == -1 {
		t.Errorf("Not all expected sections found in output: %s", outputStr)
	}

	if !(backupPos < checkPos && checkPos < snapshotsPos && snapshotsPos < forgetPos) {
		t.Errorf("Actions not in chronological order. Positions: backup=%d, check=%d, snapshots=%d, forget=%d\nOutput:\n%s",
			backupPos, checkPos, snapshotsPos, forgetPos, outputStr)
	}

	// Verify specific content
	if !contains(outputStr, "Files: 5 new, 2 changed, 100 unmodified") {
		t.Errorf("Expected backup summary not found in output: %s", outputStr)
	}
	if !contains(outputStr, "✅ check") {
		t.Errorf("Expected check heading not found in output: %s", outputStr)
	}
	if !contains(outputStr, "PASSED") {
		t.Errorf("Expected check status not found in output: %s", outputStr)
	}
	if !contains(outputStr, "✅ snapshots") {
		t.Errorf("Expected snapshots heading not found in output: %s", outputStr)
	}
	if !contains(outputStr, "Repository Snapshots: 1") {
		t.Errorf("Expected snapshots count not found in output: %s", outputStr)
	}
	if !contains(outputStr, "Date & Time          |      New | Modified |  Total Files |   Added Size |   Total Size") {
		t.Errorf("Expected snapshot table header not found in output: %s", outputStr)
	}
	if !contains(outputStr, "-------------------- | -------- | -------- | ------------ | ------------ | ------------") {
		t.Errorf("Expected snapshot table separator not found in output: %s", outputStr)
	}
	if !contains(outputStr, "✅ forget") {
		t.Errorf("Expected forget heading not found in output: %s", outputStr)
	}
	if !contains(outputStr, "1 snapshots removed") {
		t.Errorf("Expected forget summary not found in output: %s", outputStr)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
