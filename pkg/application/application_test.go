package application

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"oberfeld.be/hilfmal/pkg/datasource"
	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/planner"
	"oberfeld.be/hilfmal/pkg/publisher"
)

func TestAddDateIntervalIncludesWeeksAndDays(t *testing.T) {
	app := newTestApplication(t, successfulSource(), &fakeMailer{}, &fakePublisher{}, 7, "P1W2D")
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	got := app.gapInterval.addTo(start)
	if want := start.AddDate(0, 0, 9); !got.Equal(want) {
		t.Fatalf("gap interval = %s, want %s", got, want)
	}
}

// @lat: [[tests#Tests#Daily application workflow#Positive gap interval validation]]
func TestNewApplicationRejectsZeroGapInterval(t *testing.T) {
	_, err := NewApplication(
		successfulSource(), &fakeMailer{}, &fakePublisher{},
		"sender@example.test", "emergency@example.test",
		"reminder", "admin", "reminder", "admin",
		7, "P0D",
	)
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("NewApplication() error = %v, want positive gap interval error", err)
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

// @lat: [[tests#Tests#Daily application workflow#Publication-only mode]]
func TestPublishOnly(t *testing.T) {
	tests := []struct {
		name        string
		now         time.Time
		source      *fakeSource
		publisher   *fakePublisher
		wantContact string
		wantDate    string
		wantErr     string
		wantHours   int
	}{
		{
			name:        "current contact",
			now:         time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t)),
			source:      successfulSource(),
			publisher:   &fakePublisher{},
			wantContact: "current@example.test",
			wantDate:    "2026-11-01",
		},
		{
			name:        "before six uses previous service day",
			now:         time.Date(2026, 11, 2, 5, 59, 0, 0, zurich(t)),
			source:      successfulSource(),
			publisher:   &fakePublisher{},
			wantContact: "current@example.test",
			wantDate:    "2026-11-01",
		},
		{
			name:      "no contact",
			now:       time.Date(2026, 11, 2, 6, 0, 0, 0, zurich(t)),
			source:    successfulSource(),
			publisher: &fakePublisher{},
			wantDate:  "2026-11-02",
		},
		{
			name: "service day spans daylight saving transition",
			now:  time.Date(2026, 3, 29, 6, 0, 0, 0, zurich(t)),
			source: &fakeSource{
				contacts: []models.Contact{{Name: "Ada", Phone: "123", Email: "ada@example.test"}},
				events:   []models.Event{{UID: "dst", Start: "2026-03-29", End: "2026-03-30", Attendees: []string{"ada@example.test"}}},
			},
			publisher:   &fakePublisher{},
			wantContact: "ada@example.test",
			wantDate:    "2026-03-29",
			wantHours:   23,
		},
		{
			name:      "calendar error",
			now:       time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t)),
			source:    &fakeSource{eventsErr: errors.New("calendar failed")},
			publisher: &fakePublisher{},
			wantErr:   "calendar failed",
		},
		{
			name:      "contacts error",
			now:       time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t)),
			source:    &fakeSource{contactsErr: errors.New("contacts failed")},
			publisher: &fakePublisher{},
			wantErr:   "contacts failed",
		},
		{
			name:      "publication error",
			now:       time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t)),
			source:    successfulSource(),
			publisher: &fakePublisher{err: errors.New("publish failed")},
			wantErr:   "publish failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.source.adminErr = errors.New("admins must not be loaded")
			mail := &fakeMailer{}
			app := newTestApplication(t, tt.source, mail, tt.publisher, 7, "P1M")
			err := app.PublishOnly(tt.now)
			if tt.source.adminCalls != 0 {
				t.Fatal("publication-only mode loaded administrators")
			}
			if len(mail.sent) != 0 {
				t.Fatalf("publication-only mode sent %d emails", len(mail.sent))
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("PublishOnly() error = %v, want %q", err, tt.wantErr)
				}
				if tt.publisher.published {
					t.Fatal("page was published after a failure")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !tt.publisher.published {
				t.Fatal("page was not published")
			}
			if tt.wantContact == "" {
				if tt.publisher.contact != nil {
					t.Fatalf("published unexpected contact: %+v", tt.publisher.contact)
				}
			} else if tt.publisher.contact == nil || tt.publisher.contact.Email != tt.wantContact {
				t.Fatalf("published contact = %+v, want %s", tt.publisher.contact, tt.wantContact)
			}
			if tt.source.from.Format("2006-01-02 15:04") != tt.wantDate+" 00:00" ||
				!tt.source.to.Equal(tt.source.from.AddDate(0, 0, 1)) {
				t.Fatalf("calendar range = %s to %s, want Swiss service day %s", tt.source.from, tt.source.to, tt.wantDate)
			}
			if tt.wantHours != 0 && tt.source.to.Sub(tt.source.from) != time.Duration(tt.wantHours)*time.Hour {
				t.Fatalf("calendar range duration = %s, want %d hours", tt.source.to.Sub(tt.source.from), tt.wantHours)
			}
		})
	}
}

func TestPublishOnlyLogsIncompleteContactsWithoutSendingMail(t *testing.T) {
	source := successfulSource()
	source.contacts = append(source.contacts, models.Contact{Name: "Berta", LastName: "Test", Email: "berta@example.test"})
	mail := &fakeMailer{}
	app := newTestApplication(t, source, mail, &fakePublisher{}, 7, "P1M")
	var output bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(originalOutput)

	if err := app.PublishOnly(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Berta Test") || !strings.Contains(output.String(), "Telefonnummer fehlt") {
		t.Fatalf("publication log = %q, want contact warning", output.String())
	}
	if len(mail.sent) != 0 {
		t.Fatalf("publication-only mode sent %d emails", len(mail.sent))
	}
}

func TestPublishOnlyWithDAVAndFilePublisher(t *testing.T) {
	for _, tt := range []struct {
		name           string
		calendarFields string
	}{
		{name: "attendee", calendarFields: "ATTENDEE:mailto:ada@example.test\r\n"},
		{name: "title", calendarFields: "SUMMARY:Ada ist zuständig\r\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			templatePath := filepath.Join(directory, "web.html")
			if err := os.WriteFile(templatePath, []byte("<p>{{name}} {{phone}}</p>"), 0600); err != nil {
				t.Fatal(err)
			}
			outputPath := filepath.Join(directory, "index.html")
			pub, err := publisher.NewFilePublisher(outputPath, templatePath)
			if err != nil {
				t.Fatal(err)
			}
			source := datasource.NewNextcloudSource(publicationDAV{calendarFields: tt.calendarFields}, "calendar", "contacts", "admins")
			mail := &fakeMailer{}
			app, err := NewApplication(source, mail, pub, "sender@example.test", "emergency@example.test",
				"Erinnerung", "Hinweis", "Erinnerung", "Hinweis", 7, "P1M")
			if err != nil {
				t.Fatal(err)
			}
			if err := app.PublishOnly(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); err != nil {
				t.Fatal(err)
			}
			if len(mail.sent) != 0 {
				t.Fatalf("publication-only mode sent %d emails", len(mail.sent))
			}
			page, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(page) != "<p>Ada +41 79 123 45 67</p>" {
				t.Fatalf("published page = %q", page)
			}
		})
	}
}

func TestSameApplicationCanPublishOnlyAndRun(t *testing.T) {
	source := successfulSource()
	mail := &fakeMailer{}
	pub := &fakePublisher{}
	app := newTestApplication(t, source, mail, pub, 7, "P1M")
	now := time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))

	if err := app.PublishOnly(now); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 0 || !pub.published {
		t.Fatalf("publication-only run: published = %t, emails = %d", pub.published, len(mail.sent))
	}
	if code := app.Run(now); code != 0 {
		t.Fatalf("normal run returned %d", code)
	}
	if len(mail.sent) == 0 || source.adminCalls == 0 {
		t.Fatal("normal run did not send notifications or load administrators")
	}
}

func TestSameNamedContactsPublishAndNotifyAdmin(t *testing.T) {
	for _, tt := range []struct {
		name     string
		lastName string
		summary  string
	}{
		{name: "same full name", lastName: "Test", summary: "Alex Test"},
		{name: "same first name", lastName: "Anders", summary: "Alex"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := &fakeSource{
				admins: []string{"admin@example.test"},
				contacts: []models.Contact{
					{Name: "Alex", LastName: "Test", Phone: "456", Email: "z@example.test"},
					{Name: "Alex", LastName: tt.lastName, Phone: "123", Email: "a@example.test"},
				},
				events: []models.Event{{UID: "duty", Start: "2026-11-01", End: "2026-12-02", Summary: tt.summary}},
			}
			mail := &fakeMailer{}
			pub := &fakePublisher{}
			app := newTestApplication(t, source, mail, pub, 7, "P1M")
			if code := app.Run(time.Date(2026, 11, 1, 6, 0, 0, 0, zurich(t))); code != 0 {
				t.Fatalf("Run() = %d, want success", code)
			}
			if !pub.published || pub.contact == nil || pub.contact.Email != "a@example.test" || pub.contact.Phone != "123" {
				t.Fatalf("published contact = %#v", pub.contact)
			}
			for _, message := range mail.sent {
				if message.to == "admin@example.test" && strings.Contains(message.body, "a@example.test") &&
					strings.Contains(message.body, "Mehrere Kontakte") {
					return
				}
			}
			t.Fatalf("no admin warning for selected contact: %#v", mail.sent)
		})
	}
}

type publicationDAV struct {
	calendarFields string
}

func (d publicationDAV) Report(endpoint, _ string) (string, error) {
	var element, content string
	switch endpoint {
	case "calendar":
		element = "c:calendar-data"
		content = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:test\r\nDTSTART;VALUE=DATE:20261029\r\nDTEND;VALUE=DATE:20261105\r\n" + d.calendarFields + "END:VEVENT\r\nEND:VCALENDAR\r\n"
	case "contacts":
		element = "a:address-data"
		content = "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Test;Ada;;;\r\nFN:Ada Test\r\nTEL:+41 79 123 45 67\r\nEMAIL:ada@example.test\r\nEND:VCARD\r\n"
	default:
		return "", fmt.Errorf("unexpected DAV endpoint: %s", endpoint)
	}
	return fmt.Sprintf(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:response><d:propstat><d:prop><%s><![CDATA[%s]]></%s></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`, element, content, element), nil
}

type fakeSource struct {
	admins      []string
	adminErr    error
	adminCalls  int
	contacts    []models.Contact
	contactsErr error
	events      []models.Event
	eventsErr   error
	from        time.Time
	to          time.Time
}

func (s *fakeSource) Admins() ([]string, error) {
	s.adminCalls++
	return s.admins, s.adminErr
}

func (s *fakeSource) Contacts() ([]models.Contact, error) {
	return s.contacts, s.contactsErr
}

func (s *fakeSource) Events(from, to time.Time) ([]models.Event, error) {
	s.from, s.to = from, to
	return s.events, s.eventsErr
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
		contacts: []models.Contact{{Name: "Aktuell", LastName: "Test", Phone: "123", Email: "current@example.test"}, {Name: "Person", LastName: "Test", Phone: "456", Email: "person@example.test"}},
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
