package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"oberfeld.be/hilfmal/pkg/models"
)

type scenarioKey struct{}

type applicationScenario struct {
	source    *scenarioDataSource
	mailer    *scenarioMailer
	publisher *scenarioPublisher
	now       time.Time
	app       *Application
	exitCode  int
}

type scenarioDataSource struct {
	admins   []string
	adminErr error
	contacts []models.Contact
	events   []models.Event
	from     time.Time
	to       time.Time
}

func (s *scenarioDataSource) Admins() ([]string, error) {
	return s.admins, s.adminErr
}

func (s *scenarioDataSource) Contacts() ([]models.Contact, error) {
	return s.contacts, nil
}

func (s *scenarioDataSource) Events(from, to time.Time) ([]models.Event, error) {
	s.from, s.to = from, to
	return s.events, nil
}

func (s *scenarioDataSource) Issues() []string {
	return nil
}

type scenarioMessage struct {
	to      string
	subject string
	body    string
}

type scenarioMailer struct {
	messages []scenarioMessage
	fail     bool
}

func (m *scenarioMailer) Send(to, _, subject, body string) error {
	m.messages = append(m.messages, scenarioMessage{to: to, subject: subject, body: body})
	if m.fail {
		return errors.New("mail delivery failed")
	}
	return nil
}

type scenarioPublisher struct {
	contact   *models.Contact
	attempted bool
	fail      bool
}

func (p *scenarioPublisher) Publish(contact *models.Contact) error {
	p.attempted = true
	if p.fail {
		return errors.New("publication failed")
	}
	p.contact = contact
	return nil
}

// @lat: [[tests#Tests#Daily application workflow#Run publication and notice behavior]]
func TestApplicationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		Name:                "application",
		ScenarioInitializer: initializeApplicationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("Godog scenarios failed")
	}
}

func initializeApplicationScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		scenario := &applicationScenario{
			source:    &scenarioDataSource{},
			mailer:    &scenarioMailer{},
			publisher: &scenarioPublisher{},
		}
		return context.WithValue(ctx, scenarioKey{}, scenario), nil
	})

	ctx.Step(`^die Anwendung läuft am "([^"]+)" mit (\d+) Erinnerungstagen und Intervall "([^"]+)"$`, configureApplication)
	ctx.Step(`^Administratoren sind "([^"]+)"$`, setAdmins)
	ctx.Step(`^Mitwirkende sind$`, setContacts)
	ctx.Step(`^der Kalender enthält die Dienste$`, setEvents)
	ctx.Step(`^Administratoren können nicht gelesen werden$`, failToReadAdmins)
	ctx.Step(`^der Mailversand schlägt fehl$`, failMailDelivery)
	ctx.Step(`^die Veröffentlichung schlägt fehl$`, failPublication)
	ctx.Step(`^die Anwendung ausgeführt wird$`, runApplicationScenario)
	ctx.Step(`^endet der Lauf mit Status (\d+)$`, runHasExitCode)
	ctx.Step(`^die Webseite zeigt "([^"]+)"$`, pageShowsContact)
	ctx.Step(`^es wurde eine E-mail an "([^"]+)" versendet welche im Body "([^"]+)" enthält$`, emailBodyContains)
	ctx.Step(`^die E-Mail an "([^"]+)" konnte nicht versendet werden$`, emailWasNotSent)
	ctx.Step(`^es wurden keine E-Mails versendet$`, noEmailsSent)
	ctx.Step(`^die Webseite wurde nicht veröffentlicht$`, pageWasNotPublished)
	ctx.Step(`^eine Fehlerbenachrichtigung wurde an "([^"]+)" versucht$`, failureNotificationAttempted)
	ctx.Step(`^der Kalender wurde von "([^"]+)" bis "([^"]+)" gelesen$`, calendarRangeIs)
}

func scenarioFrom(ctx context.Context) (*applicationScenario, error) {
	scenario, ok := ctx.Value(scenarioKey{}).(*applicationScenario)
	if !ok {
		return nil, errors.New("application scenario is not initialized")
	}
	return scenario, nil
}

func configureApplication(ctx context.Context, now string, reminderDays int, gapInterval string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.now, err = time.Parse(time.RFC3339, now)
	if err != nil {
		return fmt.Errorf("parse run time: %w", err)
	}
	scenario.app, err = NewApplication(
		scenario.source, scenario.mailer, scenario.publisher,
		"sender@example.test", "emergency@example.test",
		"Hallo {{name}}, dein Dienst beginnt am {{date}}.",
		"Hinweise:\n{{issues}}",
		"Erinnerung", "Dienstplanhinweis",
		reminderDays, gapInterval,
	)
	return err
}

func setAdmins(ctx context.Context, emails string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.source.admins = strings.Split(emails, ",")
	return nil
}

func setContacts(ctx context.Context, table *godog.Table) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	for _, row := range table.Rows[1:] {
		scenario.source.contacts = append(scenario.source.contacts, models.Contact{
			Name:     row.Cells[0].Value,
			LastName: "Test",
			Phone:    row.Cells[1].Value,
			Email:    row.Cells[2].Value,
		})
	}
	return nil
}

func setEvents(ctx context.Context, table *godog.Table) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	for _, row := range table.Rows[1:] {
		scenario.source.events = append(scenario.source.events, models.Event{
			UID:       row.Cells[0].Value,
			Start:     row.Cells[1].Value,
			End:       row.Cells[2].Value,
			Attendees: strings.Split(row.Cells[3].Value, ","),
		})
	}
	return nil
}

func failToReadAdmins(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.source.adminErr = errors.New("address book unavailable")
	return nil
}

func failMailDelivery(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.mailer.fail = true
	return nil
}

func failPublication(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.publisher.fail = true
	return nil
}

func runApplicationScenario(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	if scenario.app == nil {
		return errors.New("application has not been configured")
	}
	scenario.exitCode = scenario.app.Run(scenario.now)
	return nil
}

func runHasExitCode(ctx context.Context, expected int) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	if scenario.exitCode != expected {
		return fmt.Errorf("application exit code = %d, want %d", scenario.exitCode, expected)
	}
	return nil
}

func pageShowsContact(ctx context.Context, name string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	if !scenario.publisher.attempted || scenario.publisher.contact == nil {
		return errors.New("application did not publish a contact")
	}
	if scenario.publisher.contact.Name != name {
		return fmt.Errorf("published contact is %q, want %q", scenario.publisher.contact.Name, name)
	}
	return nil
}

func emailBodyContains(ctx context.Context, recipient, text string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	for _, message := range scenario.mailer.messages {
		if message.to == recipient && strings.Contains(message.body, text) {
			return nil
		}
	}
	return fmt.Errorf("no email to %q contained %q in its body", recipient, text)
}

func emailWasNotSent(ctx context.Context, recipient string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	for _, message := range scenario.mailer.messages {
		if message.to == recipient && scenario.mailer.fail {
			return nil
		}
	}
	return fmt.Errorf("no failed email delivery was recorded for %q", recipient)
}

func noEmailsSent(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	if len(scenario.mailer.messages) != 0 {
		return fmt.Errorf("sent %d emails, want none", len(scenario.mailer.messages))
	}
	return nil
}

func pageWasNotPublished(ctx context.Context) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	if scenario.publisher.attempted {
		return errors.New("application attempted to publish despite a previous failure")
	}
	return nil
}

func failureNotificationAttempted(ctx context.Context, recipient string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	for _, message := range scenario.mailer.messages {
		if message.to == recipient && message.subject == "Fehler beim Hilfmal-Programmlauf" {
			return nil
		}
	}
	return fmt.Errorf("no failure notification was attempted for %q", recipient)
}

func calendarRangeIs(ctx context.Context, from, to string) error {
	scenario, err := scenarioFrom(ctx)
	if err != nil {
		return err
	}
	actualFrom, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return fmt.Errorf("parse expected start: %w", err)
	}
	actualTo, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return fmt.Errorf("parse expected end: %w", err)
	}
	if !scenario.source.from.Equal(actualFrom) || !scenario.source.to.Equal(actualTo) {
		return fmt.Errorf("calendar range = [%s, %s), want [%s, %s)",
			scenario.source.from.Format(time.RFC3339), scenario.source.to.Format(time.RFC3339), from, to)
	}
	return nil
}
