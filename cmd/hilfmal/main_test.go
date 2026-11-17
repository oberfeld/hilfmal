package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oberfeld.be/hilfmal/pkg/models"
)

func TestLoadConfigRequiresIntegerReminderDays(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{name: "missing", yaml: "dav_user: user\n"},
		{name: "string", yaml: "reminder_days: \"7\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadConfig(path); err == nil {
				t.Fatal("expected invalid reminder_days to fail")
			}
		})
	}
}

func TestValidateConfigRejectsNonHTTPSAndInvalidAddresses(t *testing.T) {
	config := validTestConfig()
	config.CalendarURL = "http://cloud.example.test/calendar"
	if err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("validateConfig() error = %v, want HTTPS endpoint error", err)
	}

	config = validTestConfig()
	config.Sender = "not-an-email"
	if err := validateConfig(config); err == nil {
		t.Fatal("expected invalid sender to fail")
	}

	config = validTestConfig()
	config.ReminderDays = -1
	if err := validateConfig(config); err == nil {
		t.Fatal("expected negative reminder_days to fail")
	}
}

func validTestConfig() *models.Config {
	return &models.Config{
		DAVUser:          "service",
		DAVPassword:      "secret",
		CalendarURL:      "https://cloud.example.test/calendar",
		ContactsURL:      "https://cloud.example.test/contacts",
		AdminsURL:        "https://cloud.example.test/admins",
		Sender:           "sender@example.test",
		EmergencyEmail:   "emergency@example.test",
		ReminderDays:     7,
		GapInterval:      "P1M",
		PublicPage:       "/var/www/html/index.html",
		WebTemplate:      "/app/templates/web.html",
		ReminderTemplate: "/app/templates/reminder.txt",
		AdminTemplate:    "/app/templates/admin.txt",
		ReminderSubject:  "Erinnerung",
		AdminSubject:     "Hinweis",
	}
}
