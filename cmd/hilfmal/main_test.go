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

func TestLoadConfigOverridesRuntimeSettingsFromEnvironment(t *testing.T) {
	for name, value := range map[string]string{
		"HILFMAL_DAV_USER":     "nextcloud-user",
		"HILFMAL_DAV_PASSWORD": "nextcloud-password",
	} {
		t.Setenv(name, value)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	yaml := `reminder_days: 7
dav_user: yaml-user
calendar_url: https://yaml.example.test/calendar
contacts_url: https://yaml.example.test/contacts
admins_url: https://yaml.example.test/admins
sender: yaml@example.test
emergency_email: emergency@example.test
public_page: /srv/public/index.html
`
	if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.DAVUser != "nextcloud-user" || config.DAVPassword != "nextcloud-password" {
		t.Fatalf("credentials = (%q, %q), want environment values", config.DAVUser, config.DAVPassword)
	}
	if config.CalendarURL != "https://yaml.example.test/calendar" {
		t.Fatalf("CalendarURL = %q, want YAML value", config.CalendarURL)
	}
	if config.ContactsURL != "https://yaml.example.test/contacts" {
		t.Fatalf("ContactsURL = %q, want YAML value", config.ContactsURL)
	}
	if config.AdminsURL != "https://yaml.example.test/admins" {
		t.Fatalf("AdminsURL = %q, want YAML value", config.AdminsURL)
	}
	if config.Sender != "yaml@example.test" || config.EmergencyEmail != "emergency@example.test" {
		t.Fatalf("email addresses = (%q, %q), want YAML values", config.Sender, config.EmergencyEmail)
	}
	if config.PublicPage != "/srv/public/index.html" {
		t.Fatalf("PublicPage = %q, want YAML value", config.PublicPage)
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
