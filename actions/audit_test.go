package actions

import (
	"testing"
	"time"
)

func TestValidateAuditConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuditConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AuditConfig{
				GrowThreshold:   20.0,
				ShrinkThreshold: 5.0,
			},
			wantErr: false,
		},
		{
			name: "negative grow threshold",
			config: &AuditConfig{
				GrowThreshold:   -1.0,
				ShrinkThreshold: 5.0,
			},
			wantErr: true,
		},
		{
			name: "negative shrink threshold",
			config: &AuditConfig{
				GrowThreshold:   20.0,
				ShrinkThreshold: -1.0,
			},
			wantErr: true,
		},
		{
			name: "valid config with email",
			config: &AuditConfig{
				GrowThreshold:   20.0,
				ShrinkThreshold: 5.0,
				NotifyEmailConfig: &NotifyEmailConfig{
					SMTPHost:     "smtp.example.com",
					SMTPPort:     587,
					SMTPUsername: "user",
					SMTPPassword: "pass",
					From:         "from@example.com",
					To:           "to@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid email config",
			config: &AuditConfig{
				GrowThreshold:   20.0,
				ShrinkThreshold: 5.0,
				NotifyEmailConfig: &NotifyEmailConfig{
					SMTPHost: "",
					From:     "from@example.com",
					To:       "to@example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuditConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuditConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditAction_checkSizeChanges(t *testing.T) {
	action := &AuditAction{
		config: &AuditConfig{
			GrowThreshold:   20.0,
			ShrinkThreshold: 5.0,
		},
	}

	// Create test snapshots
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	snapshots := []Snapshot{
		{
			Time:  baseTime.Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1000,
			},
		},
		{
			Time:  baseTime.Add(time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1200, // 20% growth - should trigger
			},
		},
		{
			Time:  baseTime.Add(2 * time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1100, // 8.3% shrink - should trigger
			},
		},
		{
			Time:  baseTime.Add(3 * time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1150, // 4.5% growth - should not trigger
			},
		},
		{
			Time:  baseTime.Format(time.RFC3339Nano),
			Paths: []string{"/path2"},
			Summary: BackupSummary{
				TotalBytesProcessed: 2000,
			},
		},
		{
			Time:  baseTime.Add(time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path2"},
			Summary: BackupSummary{
				TotalBytesProcessed: 2100, // 5% growth - should not trigger
			},
		},
	}

	violations := action.checkSizeChanges(snapshots)

	// Should have 0 violations: only compares the two most recent snapshots (1100 -> 1150 = 4.5% growth, below 20% threshold)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(violations))
		for _, v := range violations {
			t.Logf("Unexpected violation: %s - %s", v.CheckType, v.Message)
		}
	}
}

func TestAuditAction_checkSizeChanges_LatestOnly(t *testing.T) {
	action := &AuditAction{
		config: &AuditConfig{
			GrowThreshold:   10.0, // Lower threshold to trigger violation
			ShrinkThreshold: 5.0,
		},
	}

	// Create test snapshots where the most recent comparison triggers a violation
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	snapshots := []Snapshot{
		{
			Time:  baseTime.Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1000,
			},
		},
		{
			Time:  baseTime.Add(time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1100, // This comparison should be ignored
			},
		},
		{
			Time:  baseTime.Add(2 * time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1200, // 9.1% growth from 1100 - should not trigger (below 10%)
			},
		},
		{
			Time:  baseTime.Add(3 * time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
			Summary: BackupSummary{
				TotalBytesProcessed: 1330, // 10.8% growth from 1200 - should trigger (above 10%)
			},
		},
	}

	violations := action.checkSizeChanges(snapshots)

	// Should have 1 violation: comparing the two most recent (1200 -> 1330 = 10.8% growth)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
		for _, v := range violations {
			t.Logf("Violation: %s - %s", v.CheckType, v.Message)
		}
	}

	if len(violations) > 0 {
		v := violations[0]
		if v.CheckType != "size_growth" {
			t.Errorf("Expected size_growth, got %s", v.CheckType)
		}
		if v.Details["change_percent"] != "10.8" {
			t.Errorf("Expected change percent 10.8, got %s", v.Details["change_percent"])
		}
	}
}

func TestAuditAction_checkRetentionPolicy(t *testing.T) {
	action := &AuditAction{
		config: &AuditConfig{
			KeepDaily: 2, // Keep only 2 daily snapshots
		},
	}

	// Create test snapshots spanning multiple days
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	snapshots := []Snapshot{
		// Day 1: 3 snapshots (should trigger violation)
		{
			Time:  baseTime.Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		{
			Time:  baseTime.Add(time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		{
			Time:  baseTime.Add(2 * time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		// Day 2: 2 snapshots (should not trigger)
		{
			Time:  baseTime.AddDate(0, 0, 1).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		{
			Time:  baseTime.AddDate(0, 0, 1).Add(time.Hour).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		// Day 3: 1 snapshot (should not trigger)
		{
			Time:  baseTime.AddDate(0, 0, 2).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
	}

	violations := action.checkRetentionPolicy(snapshots)

	// Should have 1 violation for daily retention
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}

	if len(violations) > 0 {
		v := violations[0]
		if v.CheckType != "retention_daily" {
			t.Errorf("Expected check type retention_daily, got %s", v.CheckType)
		}
		if v.Details["actual"] != "3" {
			t.Errorf("Expected actual count 3, got %s", v.Details["actual"])
		}
		if v.Details["expected"] != "2" {
			t.Errorf("Expected expected count 2, got %s", v.Details["expected"])
		}
	}
}

func TestAuditAction_checkRetentionPolicy_Weekly(t *testing.T) {
	action := &AuditAction{
		config: &AuditConfig{
			KeepWeekly: 1, // Keep only 1 weekly snapshot
		},
	}

	// Create test snapshots spanning multiple weeks
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // Wednesday
	snapshots := []Snapshot{
		// Week 1 (starting Monday Dec 30, 2024)
		{
			Time:  baseTime.Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		// Week 2 (starting Monday Jan 6, 2025)
		{
			Time:  baseTime.AddDate(0, 0, 7).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
		// Week 3 (starting Monday Jan 13, 2025)
		{
			Time:  baseTime.AddDate(0, 0, 14).Format(time.RFC3339Nano),
			Paths: []string{"/path1"},
		},
	}

	violations := action.checkRetentionPolicy(snapshots)

	// Should have 1 violation for weekly retention (3 weeks > 1)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}

	if len(violations) > 0 {
		v := violations[0]
		if v.CheckType != "retention_weekly" {
			t.Errorf("Expected check type retention_weekly, got %s", v.CheckType)
		}
		if v.Details["actual"] != "3" {
			t.Errorf("Expected actual count 3, got %s", v.Details["actual"])
		}
	}
}

func TestAuditAction_checkSizeChanges_EdgeCases(t *testing.T) {
	action := &AuditAction{
		config: &AuditConfig{
			GrowThreshold:   20.0,
			ShrinkThreshold: 5.0,
		},
	}

	t.Run("single snapshot", func(t *testing.T) {
		snapshots := []Snapshot{
			{
				Time:  time.Now().Format(time.RFC3339Nano),
				Paths: []string{"/path1"},
				Summary: BackupSummary{
					TotalBytesProcessed: 1000,
				},
			},
		}

		violations := action.checkSizeChanges(snapshots)
		if len(violations) != 0 {
			t.Errorf("Expected no violations for single snapshot, got %d", len(violations))
		}
	})

	t.Run("zero size previous", func(t *testing.T) {
		snapshots := []Snapshot{
			{
				Time:  time.Now().Format(time.RFC3339Nano),
				Paths: []string{"/path1"},
				Summary: BackupSummary{
					TotalBytesProcessed: 0,
				},
			},
			{
				Time:  time.Now().Add(time.Hour).Format(time.RFC3339Nano),
				Paths: []string{"/path1"},
				Summary: BackupSummary{
					TotalBytesProcessed: 1000,
				},
			},
		}

		violations := action.checkSizeChanges(snapshots)
		if len(violations) != 0 {
			t.Errorf("Expected no violations when previous size is 0, got %d", len(violations))
		}
	})
}
