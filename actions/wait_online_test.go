package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitOnlineAction(t *testing.T) {
	// Create a test server that initially fails, then succeeds
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	waitConfig := &WaitOnlineConfig{
		URL:          server.URL,
		Timeout:      10 * time.Second,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	action := NewWaitOnlineAction(waitConfig)

	start := time.Now()
	err := action.Execute([]string{})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Should have taken some time due to retries
	if duration < 20*time.Millisecond {
		t.Errorf("Expected action to take at least 20ms, took %v", duration)
	}

	// Should have made at least 3 calls
	if callCount < 3 {
		t.Errorf("Expected at least 3 HTTP calls, got %d", callCount)
	}
}

func TestWaitOnlineActionTimeout(t *testing.T) {
	// Create a server that always returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	waitConfig := &WaitOnlineConfig{
		URL:          server.URL,
		Timeout:      100 * time.Millisecond,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
	}

	action := NewWaitOnlineAction(waitConfig)

	start := time.Now()
	err := action.Execute([]string{})
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Should have timed out within reasonable bounds
	if duration > 200*time.Millisecond {
		t.Errorf("Expected timeout within 200ms, took %v", duration)
	}
}

func TestWaitOnlineActionWithArguments(t *testing.T) {
	waitConfig := &WaitOnlineConfig{
		URL:          "http://example.com",
		Timeout:      1 * time.Second,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	action := NewWaitOnlineAction(waitConfig)

	// Test with arguments (should fail)
	err := action.Execute([]string{"arg"})
	if err == nil {
		t.Error("Expected error for arguments, got nil")
	}
}

func TestValidateWaitOnlineConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *WaitOnlineConfig
		check  func(*WaitOnlineConfig) bool
	}{
		{
			name:   "explicit config",
			config: &WaitOnlineConfig{URL: "https://custom.com", Timeout: 3 * time.Minute},
			check: func(c *WaitOnlineConfig) bool {
				return c.URL == "https://custom.com" && c.Timeout == 3*time.Minute
			},
		},
		{
			name:   "defaults applied",
			config: &WaitOnlineConfig{},
			check: func(c *WaitOnlineConfig) bool {
				return c.URL == "https://www.google.com" &&
					c.Timeout == 5*time.Minute &&
					c.InitialDelay == 1*time.Second &&
					c.MaxDelay == 30*time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWaitOnlineConfig(tt.config)
			if err != nil {
				t.Errorf("ValidateWaitOnlineConfig() error = %v", err)
			}
			if !tt.check(tt.config) {
				t.Errorf("ValidateWaitOnlineConfig() config check failed: %+v", tt.config)
			}
		})
	}
}
