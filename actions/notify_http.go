package actions

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// NotifyHTTPConfig holds configuration for HTTP notifications
type NotifyHTTPConfig struct {
	URL string
}

// ValidateNotifyHTTPConfig validates the HTTP notification config
func ValidateNotifyHTTPConfig(cfg *NotifyHTTPConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

type NotifyHTTPAction struct {
	*BaseAction
	config *NotifyHTTPConfig
}

func NewNotifyHTTPAction(cfg *NotifyHTTPConfig) *NotifyHTTPAction {
	return &NotifyHTTPAction{
		BaseAction: NewBaseAction("notify-http"),
		config:     cfg,
	}
}

func (a *NotifyHTTPAction) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("notify-http requires exactly one argument: the path to the log directory")
	}

	logDir := args[0]

	_, overallSuccess, err := analyzeBackupResults(logDir)
	if err != nil {
		return err
	}

	// Modify URL based on success/failure
	url := a.config.URL
	if !overallSuccess {
		url = strings.TrimSuffix(url, "/") + "/fail"
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request to %s failed with status code: %d", url, resp.StatusCode)
	}

	fmt.Printf("HTTP notification sent successfully (status: %d) to %s\n", resp.StatusCode, url)
	return nil
}

func NewNotifyHTTPCmd() *cobra.Command {
	var url string

	cmd := &cobra.Command{
		Use:   "notify-http [log-directory]",
		Short: "Send an HTTP notification",
		Long:  `Send an HTTP GET request to the configured URL. Appends "/fail" to the URL if the backup sequence failed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpConfig := &NotifyHTTPConfig{
				URL: url,
			}

			if err := ValidateNotifyHTTPConfig(httpConfig); err != nil {
				return fmt.Errorf("invalid HTTP config: %w", err)
			}

			action := NewNotifyHTTPAction(httpConfig)
			return action.Execute(args)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "HTTP URL to send the notification to (required)")
	cmd.MarkFlagRequired("url")

	return cmd
}
