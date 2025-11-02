package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

	// Create corresponding .out files
	os.WriteFile(filepath.Join(tmpDir, "backup.test.out"), []byte(`{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":100}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.out"), []byte(`{"message_type":"summary","num_errors":0}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.out"), []byte(`[]`), 0644)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Build the binary
	binaryPath := filepath.Join(os.TempDir(), "restic-kit-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
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
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
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
