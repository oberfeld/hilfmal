package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"oberfeld.be/hilfmal/pkg/application"
	"oberfeld.be/hilfmal/pkg/datasource"
	"oberfeld.be/hilfmal/pkg/mailer"
	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/publisher"
)

func newPublicPage() (string, func(), error) {
	publicDir, err := os.MkdirTemp("", "hilfmal-public-")
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(publicDir, "index.html"), func() {
		_ = os.RemoveAll(publicDir)
	}, nil
}

func verifyInitialRun(paths davPaths, pagePath string, today, fixedTime time.Time) error {
	if err := clearMailpitMessages(); err != nil {
		return err
	}
	if err := runAndExpect(newSource(paths.calendar, paths.contacts, paths.admins), pagePath, fixedTime, 0, "first application run"); err != nil {
		return err
	}
	if err := verifyPublicPage(pagePath, []string{"Bea", "222"}, []string{"Ada", "@example.test"}); err != nil {
		return fmt.Errorf("initial publication: %w", err)
	}
	if err := verifyMailpitMessage("cara@example.test", today.AddDate(0, 0, 7).Format("2006-01-02")); err != nil {
		return err
	}
	return verifyMailpitMessage("admin@example.test", today.AddDate(0, 0, 15).Format("2006-01-02"))
}

func verifyContactRefresh(paths davPaths, pagePath string, fixedTime time.Time) error {
	updatedContact := testContact{uid: "bea", name: "Bea", phone: "999", email: "bea@example.test"}
	if err := updateCard(paths.contacts, updatedContact); err != nil {
		return err
	}
	if err := verifyPublicPage(pagePath, []string{"222"}, []string{"999"}); err != nil {
		return fmt.Errorf("contact change appeared before the next run: %w", err)
	}
	if err := clearMailpitMessages(); err != nil {
		return err
	}
	if err := runAndExpect(newSource(paths.calendar, paths.contacts, paths.admins), pagePath, fixedTime, 0, "second application run"); err != nil {
		return err
	}
	return verifyPublicPage(pagePath, []string{"999"}, []string{"222"})
}

func verifyProductionEntryPoint(paths davPaths, pagePath string) error {
	if err := runCLI(pagePath, paths.calendar, paths.contacts, paths.admins); err != nil {
		return err
	}
	return verifyPublicPage(pagePath, []string{"999"}, nil)
}

func verifyFailureNotifications(paths davPaths, pagePath string, fixedTime time.Time) error {
	if err := clearMailpitMessages(); err != nil {
		return err
	}
	broken := newSource(paths.calendarBase+"nicht-vorhanden/", paths.contacts, paths.admins)
	if err := runAndExpect(broken, pagePath, fixedTime, 1, "broken calendar access"); err != nil {
		return err
	}
	if err := verifyMailpitMessage("admin@example.test", "fehlgeschlagen"); err != nil {
		return err
	}
	if err := verifyMailpitMessage("notfall@example.test", "fehlgeschlagen"); err != nil {
		return err
	}
	if err := verifyPublicPage(pagePath, []string{"999"}, nil); err != nil {
		return err
	}

	if err := clearMailpitMessages(); err != nil {
		return err
	}
	missingAdmins := newSource(paths.calendar, paths.contacts, paths.addressBase+"nicht-vorhanden/")
	if err := runAndExpect(missingAdmins, pagePath, fixedTime, 1, "broken admin address book"); err != nil {
		return err
	}
	if err := verifyMailpitMessage("notfall@example.test", "fehlgeschlagen"); err != nil {
		return err
	}
	return verifyPublicPage(pagePath, []string{"999"}, nil)
}

func runAndExpect(source *datasource.NextcloudSource, pagePath string, now time.Time, expectedCode int, name string) error {
	code, err := runApplication(source, pagePath, now)
	if err != nil {
		return err
	}
	if code != expectedCode {
		return fmt.Errorf("%s returned exit code %d, want %d", name, code, expectedCode)
	}
	return nil
}

func verifyPublicPage(path string, required, forbidden []string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, value := range required {
		if !strings.Contains(string(content), value) {
			return fmt.Errorf("published page does not contain %q", value)
		}
	}
	for _, value := range forbidden {
		if strings.Contains(string(content), value) {
			return fmt.Errorf("published page unexpectedly contains %q", value)
		}
	}
	return nil
}

func runApplication(source *datasource.NextcloudSource, pagePath string, now time.Time) (int, error) {
	reminderTemplate, err := os.ReadFile("/app/templates/reminder.txt")
	if err != nil {
		return 0, fmt.Errorf("read reminder template: %w", err)
	}
	adminTemplate, err := os.ReadFile("/app/templates/admin.txt")
	if err != nil {
		return 0, fmt.Errorf("read admin template: %w", err)
	}
	pub, err := publisher.NewFilePublisher(pagePath, "/app/templates/web.html")
	if err != nil {
		return 0, fmt.Errorf("create publisher: %w", err)
	}
	app, err := application.NewApplication(source,
		mailer.NewSMTPMailer("mailpit", "1025", "", ""),
		pub, sender, "notfall@example.test",
		string(reminderTemplate), string(adminTemplate),
		"Erinnerung", "Dienstplan", 7, "P1M",
	)
	if err != nil {
		return 0, fmt.Errorf("create application: %w", err)
	}
	return app.Run(now), nil
}

func newSource(calendarPath, contactsPath, adminsPath string) *datasource.NextcloudSource {
	transport := datasource.NewHTTPDAVTransport(davUser, davPass)
	return datasource.NewNextcloudSource(transport, cloudURL+calendarPath, cloudURL+contactsPath, cloudURL+adminsPath)
}

func runCLI(pagePath, calendarPath, contactsPath, adminsPath string) error {
	config := models.Config{
		DAVUser:          davUser,
		DAVPassword:      davPass,
		CalendarURL:      cloudURL + calendarPath,
		ContactsURL:      cloudURL + contactsPath,
		AdminsURL:        cloudURL + adminsPath,
		Sender:           sender,
		EmergencyEmail:   "notfall@example.test",
		ReminderDays:     7,
		GapInterval:      "P1M",
		PublicPage:       pagePath,
		WebTemplate:      "/app/templates/web.html",
		ReminderTemplate: "/app/templates/reminder.txt",
		AdminTemplate:    "/app/templates/admin.txt",
		ReminderSubject:  "Erinnerung",
		AdminSubject:     "Dienstplan",
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	configFile, err := os.CreateTemp("", "hilfmal-config-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(configFile.Name())
	if _, err := configFile.Write(data); err != nil {
		_ = configFile.Close()
		return err
	}
	if err := configFile.Close(); err != nil {
		return err
	}
	command := exec.Command("/app/hilfmal", configFile.Name())
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("production entry point failed: %w: %s", err, output)
	}
	return nil
}
