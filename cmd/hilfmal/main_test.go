package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

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
sendmail_path: /opt/mail/sendmail
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
	if config.SendmailPath != "/opt/mail/sendmail" {
		t.Fatalf("SendmailPath = %q, want YAML value", config.SendmailPath)
	}
}

func TestPublishOnlyStartupFailureDoesNotSendEmail(t *testing.T) {
	for _, tt := range []struct {
		name        string
		user        string
		password    string
		missingFile string
		wantLog     string
	}{
		{name: "missing credentials", wantLog: "dav_user"},
		{name: "missing web template", user: "user", password: "password", missingFile: "web", wantLog: "read web template"},
		{name: "missing reminder template", user: "user", password: "password", missingFile: "reminder", wantLog: "read reminder template"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			marker := filepath.Join(directory, "mail-sent")
			sendmail := filepath.Join(directory, "sendmail")
			if err := os.WriteFile(sendmail, []byte("#!/bin/sh\nprintf sent > \"$MAIL_MARKER\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", directory)
			t.Setenv("MAIL_MARKER", marker)
			t.Setenv("HILFMAL_DAV_USER", tt.user)
			t.Setenv("HILFMAL_DAV_PASSWORD", tt.password)
			path := filepath.Join(directory, "config.yaml")
			config := validTestConfig()
			config.PublicPage = filepath.Join(directory, "index.html")
			config.WebTemplate = filepath.Join(directory, "web.html")
			config.ReminderTemplate = filepath.Join(directory, "reminder.txt")
			config.AdminTemplate = filepath.Join(directory, "admin.txt")
			for name, templatePath := range map[string]string{
				"web": config.WebTemplate, "reminder": config.ReminderTemplate, "admin": config.AdminTemplate,
			} {
				if name != tt.missingFile {
					if err := os.WriteFile(templatePath, []byte("template"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			configYAML, err := yaml.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, configYAML, 0600); err != nil {
				t.Fatal(err)
			}

			var output bytes.Buffer
			originalOutput := log.Writer()
			log.SetOutput(&output)
			defer log.SetOutput(originalOutput)
			if code := runCLI([]string{"--publish-only", path}); code != 1 {
				t.Fatalf("runCLI() = %d, want failure", code)
			}
			if !strings.Contains(output.String(), tt.wantLog) {
				t.Fatalf("startup log = %q, want %q", output.String(), tt.wantLog)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("publication-only startup must not send email; marker error = %v", err)
			}
		})
	}
}

func TestNormalStartupFailureStillSendsEmail(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "mail-sent")
	sendmailPath := filepath.Join(directory, "configured-sendmail")
	if err := os.WriteFile(sendmailPath, []byte("#!/bin/sh\nprintf sent > \"$MAIL_MARKER\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	t.Setenv("MAIL_MARKER", marker)
	t.Setenv("HILFMAL_DAV_USER", "")
	t.Setenv("HILFMAL_DAV_PASSWORD", "")
	config := validTestConfig()
	config.SendmailPath = sendmailPath
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(path, configYAML, 0600); err != nil {
		t.Fatal(err)
	}
	if code := runCLI([]string{path}); code != 1 {
		t.Fatalf("runCLI() = %d, want failure", code)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("normal startup did not send an error notification: %v", err)
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

	config = validTestConfig()
	config.SendmailPath = "bin/sendmail"
	if err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "sendmail_path") {
		t.Fatalf("validateConfig() error = %v, want absolute sendmail path error", err)
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
