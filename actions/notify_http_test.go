package actions

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNotifyHTTPAction(t *testing.T) {
	// Create a temporary directory with successful exit codes
	tmpDir, err := os.MkdirTemp("", "http-test*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create successful exitcode files
	os.WriteFile(filepath.Join(tmpDir, "backup.test.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.exitcode"), []byte("0"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.exitcode"), []byte("0"), 0644)

	// Create corresponding .out files with minimal content
	os.WriteFile(filepath.Join(tmpDir, "backup.test.out"), []byte(`{"message_type":"summary","files_new":0,"files_changed":0,"files_unmodified":100}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "check.out"), []byte(`{"message_type":"summary","num_errors":0}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "snapshots.out"), []byte(`[]`), 0644)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpConfig := &NotifyHTTPConfig{
		URL: server.URL,
	}

	action := NewNotifyHTTPAction(httpConfig)

	// Test successful request
	err = action.Execute([]string{tmpDir})
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Test with no arguments (should fail)
	err = action.Execute([]string{})
	if err == nil {
		t.Error("Expected error for no arguments, got nil")
	}

	// Test with too many arguments (should fail)
	err = action.Execute([]string{tmpDir, "extra"})
	if err == nil {
		t.Error("Expected error for too many arguments, got nil")
	}
}

func TestNotifyHTTPActionFailure(t *testing.T) {
	// Create a temporary directory with successful exit codes
	tmpDir, err := os.MkdirTemp("", "http-fail-test*")
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

	// Test with invalid URL (use a non-routable IP)
	httpConfig := &NotifyHTTPConfig{
		URL: "http://192.0.2.1", // RFC 5737 test IP that should not be routable
	}

	action := NewNotifyHTTPAction(httpConfig)

	err = action.Execute([]string{tmpDir})
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestValidateNotifyHTTPConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *NotifyHTTPConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &NotifyHTTPConfig{URL: "https://example.com/notify"},
			wantErr: false,
		},
		{
			name:    "missing url",
			config:  &NotifyHTTPConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotifyHTTPConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNotifyHTTPConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
