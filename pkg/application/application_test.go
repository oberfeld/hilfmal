package application

import (
	"errors"
	"strings"
	"testing"
	"time"

	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/planner"
)

func TestRunSendsReminderAndGapNoticeBeforePublishing(t *testing.T) {
	source := &fakeSource{
		admins:   []string{"admin@example.test"},
		contacts: []models.Contact{{Name: "Aktuell", Phone: "123", Email: "current@example.test"}, {Name: "Person", Phone: "456", Email: "person@example.test"}},
		events: []models.Event{
			{UID: "today", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"current@example.test"}},
			{UID: "reminder", Start: "2026-11-08", End: "2026-11-09", Attendees: []string{"person@example.test"}},
		},
	}
	mailer := &fakeMailer{}
	publisher := &fakePublisher{}
	app := newTestApplication(t, source, mailer, publisher, 7, "P1M")

	if got := app.Run(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); got != 0 {
		t.Fatalf("Run() = %d, want 0", got)
	}
	if !publisher.published || publisher.contact == nil || publisher.contact.Name != "Aktuell" {
		t.Fatalf("published contact = %#v", publisher.contact)
	}
	if len(mailer.sent) != 2 {
		t.Fatalf("sent %d messages, want reminder and admin notice", len(mailer.sent))
	}
	if mailer.sent[0].to != "person@example.test" || !strings.Contains(mailer.sent[0].body, "2026-11-08") {
		t.Fatalf("unexpected reminder: %#v", mailer.sent[0])
	}
	if mailer.sent[1].to != "admin@example.test" || !strings.Contains(mailer.sent[1].body, "2026-11-02") {
		t.Fatalf("unexpected admin notice: %#v", mailer.sent[1])
	}
	if !source.from.Equal(time.Date(2026, 10, 31, 0, 0, 0, 0, zurich(t))) ||
		!source.to.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, zurich(t))) {
		t.Fatalf("loaded event range = [%s, %s)", source.from, source.to)
	}
}

func TestRunUsesEmergencyAddressWhenAdminReadFails(t *testing.T) {
	source := &fakeSource{adminErr: errors.New("unavailable")}
	mailer := &fakeMailer{}
	app := newTestApplication(t, source, mailer, &fakePublisher{}, 7, "P1M")

	if got := app.Run(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); got != 1 {
		t.Fatalf("Run() = %d, want 1", got)
	}
	if len(mailer.sent) != 1 || mailer.sent[0].to != "emergency@example.test" {
		t.Fatalf("failure notifications = %#v", mailer.sent)
	}
}

func TestRunDoesNotPublishAfterMailFailure(t *testing.T) {
	mailer := &fakeMailer{err: errors.New("mail failed")}
	publisher := &fakePublisher{}
	app := newTestApplication(t, successfulSource(), mailer, publisher, 7, "P1M")

	if got := app.Run(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); got != 1 {
		t.Fatalf("Run() = %d, want 1", got)
	}
	if publisher.published {
		t.Fatal("published despite mail failure")
	}
}

func TestRunReportsPublisherFailureToAdminsAndEmergency(t *testing.T) {
	mailer := &fakeMailer{}
	publisher := &fakePublisher{err: errors.New("publish failed")}
	app := newTestApplication(t, successfulSource(), mailer, publisher, 7, "P1M")

	if got := app.Run(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); got != 1 {
		t.Fatalf("Run() = %d, want 1", got)
	}
	if len(mailer.sent) < 3 {
		t.Fatalf("expected regular messages and two failure notifications, got %d", len(mailer.sent))
	}
	last := mailer.sent[len(mailer.sent)-2:]
	if last[0].to != "admin@example.test" || last[1].to != "emergency@example.test" {
		t.Fatalf("failure recipients = %#v", last)
	}
}

func TestMonthRangeUsesSwissLocalMidnight(t *testing.T) {
	app := newTestApplication(t, successfulSource(), &fakeMailer{}, &fakePublisher{}, 0, "P1M")
	today, target, gapEnd, err := app.calculateDateRange(
		time.Date(2026, 10, 31, 6, 0, 0, 0, zurich(t)),
		planner.NewPlanner(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if today.Format(time.RFC3339) != "2026-10-31T00:00:00+01:00" {
		t.Fatalf("service date start = %s", today.Format(time.RFC3339))
	}
	if target.Format("2006-01-02") != today.Format("2006-01-02") {
		t.Fatalf("target = %s, expected today", target)
	}
	if gapEnd.Format("2006-01-02 15:04") != "2026-12-01 00:00" {
		t.Fatalf("gap end = %s", gapEnd)
	}
}

type fakeSource struct {
	admins   []string
	adminErr error
	contacts []models.Contact
	events   []models.Event
	from     time.Time
	to       time.Time
}

func (s *fakeSource) Admins() ([]string, error) {
	return s.admins, s.adminErr
}

func (s *fakeSource) Contacts() ([]models.Contact, error) {
	return s.contacts, nil
}

func (s *fakeSource) Events(from, to time.Time) ([]models.Event, error) {
	s.from, s.to = from, to
	return s.events, nil
}

func (s *fakeSource) Issues() []string {
	return nil
}

type sentMessage struct {
	to      string
	from    string
	subject string
	body    string
}

type fakeMailer struct {
	sent []sentMessage
	err  error
}

func (m *fakeMailer) Send(to, from, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sentMessage{to: to, from: from, subject: subject, body: body})
	return nil
}

type fakePublisher struct {
	contact   *models.Contact
	published bool
	err       error
}

func (p *fakePublisher) Publish(contact *models.Contact) error {
	if p.err != nil {
		return p.err
	}
	p.contact, p.published = contact, true
	return nil
}

func newTestApplication(t *testing.T, source *fakeSource, mailer *fakeMailer, publisher *fakePublisher, reminderDays int, gapInterval string) *Application {
	t.Helper()
	app, err := NewApplication(
		source, mailer, publisher,
		"sender@example.test", "emergency@example.test",
		"Hallo {{name}}, dein Dienst beginnt am {{date}}.",
		"Hinweise:\n{{issues}}",
		"Erinnerung", "Dienstplanhinweis",
		reminderDays, gapInterval,
	)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func successfulSource() *fakeSource {
	return &fakeSource{
		admins:   []string{"admin@example.test"},
		contacts: []models.Contact{{Name: "Aktuell", Phone: "123", Email: "current@example.test"}, {Name: "Person", Phone: "456", Email: "person@example.test"}},
		events: []models.Event{
			{UID: "today", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"current@example.test"}},
			{UID: "reminder", Start: "2026-11-08", End: "2026-11-09", Attendees: []string{"person@example.test"}},
		},
	}
}

func zurich(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}
	return location
}
