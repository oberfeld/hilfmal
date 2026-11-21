package main

import (
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"oberfeld.be/hilfmal/pkg/application"
	"oberfeld.be/hilfmal/pkg/datasource"
	"oberfeld.be/hilfmal/pkg/mailer"
	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/publisher"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s /path/to/config.yaml\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		log.Printf("Hilfmal: Startup failed: %v", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Printf("Hilfmal: Startup failed: %v", err)
		notifyError(config.Sender, config.EmergencyEmail, "Fehler beim Hilfmal-Programmstart")
		os.Exit(1)
	}

	// Load templates
	reminderTemplate, err := os.ReadFile(config.ReminderTemplate)
	if err != nil {
		log.Printf("Hilfmal: Startup failed: failed to read reminder template: %v", err)
		notifyError(config.Sender, config.EmergencyEmail, "Fehler beim Hilfmal-Programmstart")
		os.Exit(1)
	}

	adminTemplate, err := os.ReadFile(config.AdminTemplate)
	if err != nil {
		log.Printf("Hilfmal: Startup failed: failed to read admin template: %v", err)
		notifyError(config.Sender, config.EmergencyEmail, "Fehler beim Hilfmal-Programmstart")
		os.Exit(1)
	}

	// Set up data source
	transport := datasource.NewHTTPDAVTransport(config.DAVUser, config.DAVPassword)
	source := datasource.NewNextcloudSource(transport, config.CalendarURL, config.ContactsURL, config.AdminsURL)

	// Set up mailer
	mail := mailer.NewSystemMailer("sendmail")

	// Set up publisher
	pub, err := publisher.NewFilePublisher(config.PublicPage, config.WebTemplate)
	if err != nil {
		log.Printf("Hilfmal: Startup failed: %v", err)
		notifyError(config.Sender, config.EmergencyEmail, "Fehler beim Hilfmal-Programmstart")
		os.Exit(1)
	}

	// Create application
	app, err := application.NewApplication(
		source,
		mail,
		pub,
		config.Sender,
		config.EmergencyEmail,
		string(reminderTemplate),
		string(adminTemplate),
		config.ReminderSubject,
		config.AdminSubject,
		config.ReminderDays,
		config.GapInterval,
	)
	if err != nil {
		log.Printf("Hilfmal: Startup failed: %v", err)
		notifyError(config.Sender, config.EmergencyEmail, "Fehler beim Hilfmal-Programmstart")
		os.Exit(1)
	}

	// Run application
	exitCode := app.Run(time.Now())
	os.Exit(exitCode)
}

func loadConfig(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found or not readable: %w", err)
	}

	var config models.Config
	var values map[string]yaml.Node
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("invalid YAML configuration: %w", err)
	}
	if values == nil {
		return nil, fmt.Errorf("configuration must be a YAML mapping")
	}
	if _, ok := values["reminder_days"]; !ok {
		return nil, fmt.Errorf("configuration missing or invalid: reminder_days")
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML configuration: %w", err)
	}
	applyEnvironmentConfig(&config)

	return &config, nil
}

func applyEnvironmentConfig(config *models.Config) {
	fields := []struct {
		name   string
		target *string
	}{
		{name: "HILFMAL_DAV_USER", target: &config.DAVUser},
		{name: "HILFMAL_DAV_PASSWORD", target: &config.DAVPassword},
	}
	for _, field := range fields {
		if value, ok := os.LookupEnv(field.name); ok {
			*field.target = value
		}
	}
}

func validateConfig(config *models.Config) error {
	required := []string{
		"dav_user", config.DAVUser,
		"dav_password", config.DAVPassword,
		"calendar_url", config.CalendarURL,
		"contacts_url", config.ContactsURL,
		"admins_url", config.AdminsURL,
		"sender", config.Sender,
		"emergency_email", config.EmergencyEmail,
		"gap_interval", config.GapInterval,
		"public_page", config.PublicPage,
		"web_template", config.WebTemplate,
		"reminder_template", config.ReminderTemplate,
		"admin_template", config.AdminTemplate,
		"reminder_subject", config.ReminderSubject,
		"admin_subject", config.AdminSubject,
	}

	for i := 0; i < len(required); i += 2 {
		key := required[i]
		value := required[i+1]
		if value == "" {
			return fmt.Errorf("configuration missing or invalid: %s", key)
		}
	}
	if config.ReminderDays < 0 {
		return fmt.Errorf("configuration missing or invalid: reminder_days")
	}
	for _, address := range []string{config.Sender, config.EmergencyEmail} {
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address || address != strings.TrimSpace(address) {
			return fmt.Errorf("configuration contains an invalid email address")
		}
	}
	for _, endpoint := range []string{config.CalendarURL, config.ContactsURL, config.AdminsURL} {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("DAV endpoints must be HTTPS URLs without embedded credentials")
		}
	}

	return nil
}

func notifyError(sender, recipient, subject string) {
	if sender == "" || recipient == "" {
		return
	}

	mail := mailer.NewSystemMailer("sendmail")
	if sendErr := mail.Send(recipient, sender, subject, "Der Hilfmal-Programmlauf konnte nicht gestartet werden. Bitte das Serverprotokoll und die Konfiguration prüfen."); sendErr != nil {
		log.Printf("Hilfmal: Startup error notification failed: %v", sendErr)
	}
}
