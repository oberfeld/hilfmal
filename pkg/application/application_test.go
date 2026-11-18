package application

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/planner"
)

func TestAddDateIntervalIncludesWeeksAndDays(t *testing.T) {
	app := newTestApplication(t, successfulSource(), &fakeMailer{}, &fakePublisher{}, 7, "P1M")
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	got, err := app.addDateInterval(start, "P1W2D")
	if err != nil {
		t.Fatal(err)
	}
	if want := start.AddDate(0, 0, 9); !got.Equal(want) {
		t.Fatalf("addDateInterval() = %s, want %s", got, want)
	}
}

func TestSendAdminNoticesReturnsMailerError(t *testing.T) {
	app := newTestApplication(t, successfulSource(), &fakeMailer{err: errors.New("mail failed")}, &fakePublisher{}, 7, "P1M")

	if err := app.sendAdminNotices([]string{"admin@example.test"}, []string{"missing duty"}); err == nil {
		t.Fatal("sendAdminNotices() returned nil after mail failure")
	}
}

func TestReportFailureLogsNotificationError(t *testing.T) {
	var output bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(originalOutput)

	app := newTestApplication(t, successfulSource(), &fakeMailer{err: errors.New("mail failed")}, &fakePublisher{}, 7, "P1M")
	app.reportFailure(errors.New("run failed"), "stage", nil)

	if !strings.Contains(output.String(), "Failed to send error notification") {
		t.Fatalf("failure log = %q, want notification delivery error", output.String())
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
