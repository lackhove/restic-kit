package shared

import (
	"fmt"

	gomail "gopkg.in/gomail.v2"
)

// NotifyEmailConfig holds configuration for email notifications
type NotifyEmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	From         string
	To           string
}

// ValidateNotifyEmailConfig validates the email notification config
func ValidateNotifyEmailConfig(cfg *NotifyEmailConfig) error {
	if cfg.SMTPHost == "" {
		return fmt.Errorf("smtp-host is required")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	if cfg.From == "" {
		return fmt.Errorf("from is required")
	}
	if cfg.To == "" {
		return fmt.Errorf("to is required")
	}
	if cfg.SMTPUsername == "" {
		return fmt.Errorf("smtp-username is required")
	}
	if cfg.SMTPPassword == "" {
		return fmt.Errorf("smtp-password is required")
	}
	return nil
}

// SendEmail sends an email with the given configuration
func SendEmail(cfg *NotifyEmailConfig, subject, body string, attachments []string, dryRun bool) error {
	if dryRun {
		fmt.Println("DRY RUN: Would send email with subject:", subject)
		fmt.Println("DRY RUN: Email body preview:")
		fmt.Println(body)
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", cfg.To)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// Attach log files
	for _, attachment := range attachments {
		m.Attach(attachment)
	}

	d := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully")
	return nil
}
