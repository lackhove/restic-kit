package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupAction(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Test successful cleanup (all exit codes 0)
	t.Run("SuccessfulCleanup", func(t *testing.T) {
		logDir := filepath.Join(tempDir, "success-logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatalf("Failed to create log dir: %v", err)
		}

		// Create successful exitcode files
		createExitCodeFile(t, logDir, "backup.etc.exitcode", 0)
		createExitCodeFile(t, logDir, "backup.docker-confs.exitcode", 0)
		createExitCodeFile(t, logDir, "check.exitcode", 0)

		// Create corresponding .out files (required by analyzeBackupResults)
		createOutFile(t, logDir, "backup.etc.out", `{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":10}`)
		createOutFile(t, logDir, "backup.docker-confs.out", `{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":20}`)
		createOutFile(t, logDir, "check.out", `{"message_type":"status","num_errors":0}`)

		action := NewCleanupAction(&CleanupConfig{})
		err := action.Execute([]string{logDir})

		if err != nil {
			t.Errorf("Expected successful cleanup, got error: %v", err)
		}

		// Check that directory was removed
		if _, err := os.Stat(logDir); !os.IsNotExist(err) {
			t.Errorf("Expected log directory to be removed, but it still exists")
		}
	})

	// Test failed cleanup (some exit codes non-zero)
	t.Run("FailedCleanup", func(t *testing.T) {
		logDir := filepath.Join(tempDir, "failure-logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatalf("Failed to create log dir: %v", err)
		}

		// Create mixed exitcode files (one failure)
		createExitCodeFile(t, logDir, "backup.etc.exitcode", 0)
		createExitCodeFile(t, logDir, "backup.docker-confs.exitcode", 1) // Failure
		createExitCodeFile(t, logDir, "check.exitcode", 0)

		// Create corresponding .out files
		createOutFile(t, logDir, "backup.etc.out", `{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":10}`)
		createOutFile(t, logDir, "backup.docker-confs.out", `{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":20}`)
		createOutFile(t, logDir, "check.out", `{"message_type":"status","num_errors":0}`)

		action := NewCleanupAction(&CleanupConfig{})
		err := action.Execute([]string{logDir})

		if err != nil {
			t.Errorf("Expected cleanup to complete (even with failures), got error: %v", err)
		}

		// Check that directory still exists
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			t.Errorf("Expected log directory to be kept for debugging, but it was removed")
		}
	})

	// Test invalid arguments
	t.Run("InvalidArgs", func(t *testing.T) {
		action := NewCleanupAction(&CleanupConfig{})

		// No arguments
		err := action.Execute([]string{})
		if err == nil {
			t.Error("Expected error for no arguments, got nil")
		}

		// Too many arguments
		err = action.Execute([]string{"dir1", "dir2"})
		if err == nil {
			t.Error("Expected error for too many arguments, got nil")
		}

		// Non-existent directory
		err = action.Execute([]string{"/non/existent/directory"})
		if err == nil {
			t.Error("Expected error for non-existent directory, got nil")
		}
	})
}

func createExitCodeFile(t *testing.T, dir, filename string, code int) {
	path := filepath.Join(dir, filename)
	content := fmt.Sprintf("%d\n", code)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create exitcode file %s: %v", path, err)
	}
}

func createOutFile(t *testing.T, dir, filename, content string) {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create out file %s: %v", path, err)
	}
}
