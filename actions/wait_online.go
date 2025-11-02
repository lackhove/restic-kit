package actions

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// WaitOnlineConfig holds configuration for waiting online
type WaitOnlineConfig struct {
	URL          string
	Timeout      time.Duration
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// ValidateWaitOnlineConfig validates the wait online config and sets defaults
func ValidateWaitOnlineConfig(cfg *WaitOnlineConfig) error {
	if cfg.URL == "" {
		cfg.URL = "https://www.google.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.InitialDelay == 0 {
		cfg.InitialDelay = 1 * time.Second
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	return nil
}

type WaitOnlineAction struct {
	*BaseAction
	config *WaitOnlineConfig
}

func NewWaitOnlineAction(cfg *WaitOnlineConfig) *WaitOnlineAction {
	return &WaitOnlineAction{
		BaseAction: NewBaseAction("wait-online"),
		config:     cfg,
	}
}

func (a *WaitOnlineAction) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("wait-online does not accept any arguments")
	}

	client := &http.Client{
		Timeout: 10 * time.Second, // 10 second timeout for each request
	}

	startTime := time.Now()
	delay := a.config.InitialDelay

	for {
		resp, err := client.Get(a.config.URL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				fmt.Printf("Successfully reached %s after %v\n", a.config.URL, time.Since(startTime))
				return nil
			}
		}

		if time.Since(startTime) >= a.config.Timeout {
			return fmt.Errorf("timeout reached: could not reach %s within %v", a.config.URL, a.config.Timeout)
		}

		fmt.Printf("Failed to reach %s, retrying in %v...\n", a.config.URL, delay)
		time.Sleep(delay)

		// Exponential backoff with max delay
		delay *= 2
		if delay > a.config.MaxDelay {
			delay = a.config.MaxDelay
		}
	}
}

func NewWaitOnlineCmd() *cobra.Command {
	var url string
	var timeout, initialDelay, maxDelay time.Duration

	cmd := &cobra.Command{
		Use:   "wait-online",
		Short: "Wait for network connectivity",
		Long:  `Wait for the configured URL to be reachable with exponential backoff.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			waitConfig := &WaitOnlineConfig{
				URL:          url,
				Timeout:      timeout,
				InitialDelay: initialDelay,
				MaxDelay:     maxDelay,
			}

			if err := ValidateWaitOnlineConfig(waitConfig); err != nil {
				return fmt.Errorf("invalid wait-online config: %w", err)
			}

			action := NewWaitOnlineAction(waitConfig)
			return action.Execute(args)
		},
	}

	cmd.Flags().StringVar(&url, "url", "https://www.google.com", "URL to check for connectivity")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Total timeout for waiting")
	cmd.Flags().DurationVar(&initialDelay, "initial-delay", 1*time.Second, "Initial delay between retries")
	cmd.Flags().DurationVar(&maxDelay, "max-delay", 30*time.Second, "Maximum delay between retries")

	return cmd
}
