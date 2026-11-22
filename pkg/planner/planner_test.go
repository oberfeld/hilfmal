package planner

import (
	"strings"
	"testing"
	"time"

	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/parser"
)

// @lat: [[tests#Tests#Daily application workflow#Swiss service date and ranges]]
func TestServiceDate(t *testing.T) {
	plannerUT := NewPlanner()

	tests := []struct {
		name     string
		now      time.Time
		expected string
	}{
		{
			name:     "After 06:00 Zurich time",
			now:      time.Date(2024, 1, 15, 10, 0, 0, 0, timeZone()),
			expected: "2024-01-15",
		},
		{
			name:     "Before 06:00 Zurich time",
			now:      time.Date(2024, 1, 15, 4, 0, 0, 0, timeZone()),
			expected: "2024-01-14",
		},
		{
			name:     "Exactly at 06:00",
			now:      time.Date(2024, 1, 15, 6, 0, 0, 0, timeZone()),
			expected: "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plannerUT.ServiceDate(tt.now)
			if err != nil {
				t.Fatal(err)
			}
			if result != tt.expected {
				t.Errorf("ServiceDate() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestServiceDateAtDaylightSavingTransitions(t *testing.T) {
	p := NewPlanner()
	location := timeZone()
	for _, day := range []string{"2026-03-29", "2026-10-25", "2026-01-01"} {
		t.Run(day, func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02", day, location)
			if err != nil {
				t.Fatal(err)
			}
			before, err := p.ServiceDate(time.Date(date.Year(), date.Month(), date.Day(), 5, 59, 59, 0, location))
			if err != nil {
				t.Fatal(err)
			}
			at, err := p.ServiceDate(time.Date(date.Year(), date.Month(), date.Day(), 6, 0, 0, 0, location))
			if err != nil {
				t.Fatal(err)
			}
			if before != date.AddDate(0, 0, -1).Format("2006-01-02") || at != day {
				t.Fatalf("service dates before/at 06:00 = %s/%s", before, at)
			}
		})
	}
}

// @lat: [[tests#Tests#Calendar and contact resolution#Priority and contact validity]]
func TestSelect(t *testing.T) {
	p := NewPlanner()

	contact1 := models.Contact{
		Name:  "John",
		Phone: "123456789",
		Email: "john@example.com",
	}

	contact2 := models.Contact{
		Name:  "Jane",
		Phone: "987654321",
		Email: "jane@example.com",
	}

	tests := []struct {
		name     string
		date     string
		events   []models.Event
		contacts []models.Contact
		expected *models.Contact
	}{
		{
			name: "Single event with valid contact",
			date: "2024-01-15",
			events: []models.Event{
				{
					UID:       "1",
					Start:     "2024-01-15",
					End:       "2024-01-16",
					Attendees: []string{"john@example.com"},
					Created:   nil,
				},
			},
			contacts: []models.Contact{contact1},
			expected: &contact1,
		},
		{
			name:     "No events",
			date:     "2024-01-15",
			events:   []models.Event{},
			contacts: []models.Contact{contact1},
			expected: nil,
		},
		{
			name: "Event outside date range",
			date: "2024-01-15",
			events: []models.Event{
				{
					UID:       "1",
					Start:     "2024-01-16",
					End:       "2024-01-17",
					Attendees: []string{"john@example.com"},
					Created:   nil,
				},
			},
			contacts: []models.Contact{contact1},
			expected: nil,
		},
		{
			name: "Multiple events, shorter has priority",
			date: "2024-01-15",
			events: []models.Event{
				{
					UID:       "1",
					Start:     "2024-01-15",
					End:       "2024-01-18", // 3 days
					Attendees: []string{"john@example.com"},
					Created:   nil,
				},
				{
					UID:       "2",
					Start:     "2024-01-15",
					End:       "2024-01-16", // 1 day (shorter)
					Attendees: []string{"jane@example.com"},
					Created:   nil,
				},
			},
			contacts: []models.Contact{contact1, contact2},
			expected: &contact2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Select(tt.date, tt.events, tt.contacts)
			if result == nil && tt.expected != nil {
				t.Errorf("Select() = nil, want %v", tt.expected)
			}
			if result != nil && tt.expected == nil {
				t.Errorf("Select() = %v, want nil", result)
			}
			if result != nil && tt.expected != nil && result.Email != tt.expected.Email {
				t.Errorf("Select() = %s, want %s", result.Email, tt.expected.Email)
			}
		})
	}
}

// @lat: [[tests#Tests#Calendar and contact resolution#Organizer fallback]]
func TestOrganizerFallback(t *testing.T) {
	contacts := []models.Contact{
		{Name: "Organizer", Phone: "123", Email: "organizer@example.test"},
		{Name: "Attendee", Phone: "456", Email: "attendee@example.test"},
		{Name: "Incomplete", Email: "incomplete@example.test"},
	}
	for _, tt := range []struct {
		name      string
		organizer string
		attendees []string
		want      string
	}{
		{name: "unknown attendee", organizer: "organizer@example.test", attendees: []string{"unknown@example.test"}, want: "organizer@example.test"},
		{name: "incomplete attendee", organizer: "organizer@example.test", attendees: []string{"incomplete@example.test"}, want: "organizer@example.test"},
		{name: "no attendee", organizer: "organizer@example.test", want: "organizer@example.test"},
		{name: "organizer also attendee", organizer: "organizer@example.test", attendees: []string{"ORGANIZER@example.test"}, want: "organizer@example.test"},
		{name: "other attendee preferred", organizer: "organizer@example.test", attendees: []string{"ORGANIZER@example.test", "attendee@example.test"}, want: "attendee@example.test"},
		{name: "unknown organizer", organizer: "unknown@example.test", attendees: []string{"attendee@example.test"}, want: "attendee@example.test"},
		{name: "no valid contact", organizer: "unknown@example.test", attendees: []string{"incomplete@example.test"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			event := models.Event{UID: "duty", Start: "2026-11-01", End: "2026-11-02", Organizer: tt.organizer, Attendees: tt.attendees}
			p := NewPlanner()
			got := p.Select("2026-11-01", []models.Event{event}, contacts)
			if tt.want == "" {
				if got != nil || len(p.Issues([]models.Event{event}, contacts)) != 1 {
					t.Fatalf("Select() = %#v, want no contact and an issue", got)
				}
			} else if got == nil || got.Email != tt.want || len(p.Issues([]models.Event{event}, contacts)) != 0 {
				t.Fatalf("Select() = %#v, want %s and no issues", got, tt.want)
			}
		})
	}

	t.Run("event priority remains first", func(t *testing.T) {
		events := []models.Event{
			{UID: "long", Start: "2026-11-01", End: "2026-11-05", Attendees: []string{"attendee@example.test"}},
			{UID: "short", Start: "2026-11-01", End: "2026-11-02", Organizer: "organizer@example.test"},
		}
		for _, order := range [][]models.Event{events, reverseEvents(events)} {
			got := NewPlanner().Select("2026-11-01", order, contacts)
			if got == nil || got.Email != "organizer@example.test" {
				t.Fatalf("Select() = %#v, want shorter event's organizer", got)
			}
		}
	})

	t.Run("calendar data and assignment", func(t *testing.T) {
		ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:duty\nDTSTART;VALUE=DATE:20261101\n" +
			"ATTENDEE:mailto:unknown@example.test\nORGANIZER;CN=Organizer:mailto:organizer@example.test\n" +
			"END:VEVENT\nEND:VCALENDAR\n"
		start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
		events, issues, err := parser.NewCalendarParser().Parse(ical, start, start.AddDate(0, 0, 1))
		if err != nil || len(issues) != 0 {
			t.Fatalf("Parse() issues = %v, error = %v", issues, err)
		}
		p := NewPlanner()
		if got := p.Select("2026-11-01", events, contacts); got == nil || got.Email != "organizer@example.test" {
			t.Fatalf("Select() = %#v, want organizer", got)
		}
		if got := p.Reminder("2026-11-01", events, contacts); got == nil || got.Email != "organizer@example.test" {
			t.Fatalf("Reminder() = %#v, want organizer", got)
		}
		if len(p.Issues(events, contacts)) != 0 || len(p.Gaps("2026-11-01", "2026-11-02", events, contacts)) != 0 {
			t.Fatal("valid organizer must not produce an issue or gap")
		}
	})
}

// @lat: [[tests#Tests#Calendar and contact resolution#Title-based contact resolution]]
func TestSummaryFallback(t *testing.T) {
	contacts := []models.Contact{
		{Name: "Ada", LastName: "Müller", Phone: "123", Email: "invalid", AdditionalEmails: []string{"ada@example.test", "ada.work@example.test"}},
		{Name: "Ada", LastName: "Meier", Phone: "456", Email: "meier@example.test"},
		{Name: "Berta", LastName: "Zeller", Phone: "789", Email: "berta@example.test"},
		{Name: "Anna", LastName: "Huber", Phone: "111", Email: "anna@example.test"},
		{Name: "Nick", LastName: "Voll", Phone: "222", Email: "nick@example.test"},
		{Name: "Nick", LastName: "Unvollständig", Email: "incomplete@example.test"},
	}
	for _, tt := range []struct {
		name      string
		summary   string
		organizer string
		attendees []string
		want      string
	}{
		{name: "unique first name", summary: "Berta 2 ist zuständig", want: "berta@example.test"},
		{name: "case insensitive surname", summary: "ADA MÜLLER ist zuständig", want: "ada@example.test"},
		{name: "other name is not a substring match", summary: "Johanna ist zuständig"},
		{name: "two surnames are ambiguous", summary: "Ada Müller und Ada Meier"},
		{name: "different first names are ambiguous", summary: "Ada und Berta"},
		{name: "surname resolves multiple first names", summary: "Ada Meier und Berta", want: "meier@example.test"},
		{name: "incomplete contact does not cause ambiguity", summary: "Nick ist zuständig", want: "nick@example.test"},
		{name: "attendee before title", summary: "Berta ist zuständig", attendees: []string{"ada.work@example.test"}, want: "ada.work@example.test"},
		{name: "title before organizer", summary: "Berta ist zuständig", organizer: "meier@example.test", attendees: []string{"MEIER@example.test"}, want: "berta@example.test"},
		{name: "organizer uses another attendee email", summary: "Berta ist zuständig", organizer: "ada@example.test", attendees: []string{"ada.work@example.test"}, want: "berta@example.test"},
		{name: "organizer when title has no name", summary: "Pikettdienst", organizer: "berta@example.test", want: "berta@example.test"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			event := models.Event{
				UID: "duty", Start: "2026-11-01", End: "2026-11-02", Summary: tt.summary,
				Organizer: tt.organizer, Attendees: tt.attendees,
			}
			p := NewPlanner()
			got := p.Select("2026-11-01", []models.Event{event}, contacts)
			if tt.want == "" {
				if got != nil || len(p.Issues([]models.Event{event}, contacts)) != 1 {
					t.Fatalf("Select() = %#v, want no contact and an issue", got)
				}
			} else if got == nil || got.Email != tt.want || len(p.Issues([]models.Event{event}, contacts)) != 0 {
				t.Fatalf("Select() = %#v, want %s and no issues", got, tt.want)
			}
		})
	}

	t.Run("ambiguous first name chooses by email regardless of surname", func(t *testing.T) {
		contacts := []models.Contact{
			{Name: "Alex", LastName: "Alex", Phone: "123", Email: "z@example.test"},
			{Name: "Alex", LastName: "Test", Phone: "456", Email: "a@example.test"},
			{Name: "ALEX", LastName: "Andere", Phone: "789", Email: "m@example.test"},
		}
		event := models.Event{UID: "duty", Start: "2026-11-01", End: "2026-11-02", Summary: "Alex ist zuständig"}
		p := NewPlanner()
		for _, order := range [][]models.Contact{contacts, {contacts[2], contacts[1], contacts[0]}} {
			got := p.Select("2026-11-01", []models.Event{event}, order)
			if got == nil || got.Email != "a@example.test" {
				t.Fatalf("Select() = %#v, want a@example.test", got)
			}
			issues := p.Issues([]models.Event{event}, order)
			if len(issues) != 1 || !strings.Contains(issues[0], "a@example.test") {
				t.Fatalf("Issues() = %v, want selected email in warning", issues)
			}
			if gaps := p.Gaps("2026-11-01", "2026-11-02", []models.Event{event}, order); len(gaps) != 0 {
				t.Fatalf("Gaps() = %v, want none", gaps)
			}
			if got := p.Reminder("2026-11-01", []models.Event{event}, order); got == nil || got.Email != "a@example.test" {
				t.Fatalf("Reminder() = %#v, want a@example.test", got)
			}
		}
		event.Summary = "Alex Alex ist zuständig"
		if got := p.Select("2026-11-01", []models.Event{event}, contacts); got == nil || got.Email != "z@example.test" {
			t.Fatalf("Select() = %#v, want z@example.test", got)
		}
		if issues := p.Issues([]models.Event{event}, contacts); len(issues) != 0 {
			t.Fatalf("Issues() = %v, want none for distinct surname", issues)
		}
	})

	t.Run("same full name selects deterministically and reports ambiguity", func(t *testing.T) {
		contacts := []models.Contact{
			{Name: "Alex", LastName: "Test", Phone: "456", Email: "z@example.test"},
			{Name: "ALEX", LastName: "test", Phone: "123", Email: "bad", AdditionalEmails: []string{"alex@example.test"}},
			{Name: "Alex", LastName: "Test", Phone: "789", Email: "other@example.test"},
		}
		event := models.Event{UID: "duty", Start: "2026-11-01", End: "2026-11-02", Summary: "Alex Test ist zuständig"}
		for _, order := range [][]models.Contact{contacts, {contacts[2], contacts[1], contacts[0]}} {
			p := NewPlanner()
			if got := p.Select("2026-11-01", []models.Event{event}, order); got == nil || got.Email != "alex@example.test" || got.Phone != "123" {
				t.Fatalf("Select() = %#v, want alex@example.test", got)
			}
			issues := p.Issues([]models.Event{event}, order)
			if len(issues) != 1 || !strings.Contains(issues[0], "alex@example.test") {
				t.Fatalf("Issues() = %v, want selected email in warning", issues)
			}
			if gaps := p.Gaps("2026-11-01", "2026-11-02", []models.Event{event}, order); len(gaps) != 0 {
				t.Fatalf("Gaps() = %v, want none", gaps)
			}
			if got := p.Reminder("2026-11-01", []models.Event{event}, order); got == nil || got.Email != "alex@example.test" {
				t.Fatalf("Reminder() = %#v, want alex@example.test", got)
			}
		}
	})

	t.Run("same name and email use phone as deterministic tie-break", func(t *testing.T) {
		contacts := []models.Contact{
			{Name: "Alex", LastName: "Test", Phone: "456", Email: "alex@example.test"},
			{Name: "Alex", LastName: "Test", Phone: "123", Email: "alex@example.test"},
		}
		event := models.Event{UID: "duty", Start: "2026-11-01", End: "2026-11-02", Summary: "Alex Test"}
		for _, order := range [][]models.Contact{contacts, {contacts[1], contacts[0]}} {
			got := NewPlanner().Select("2026-11-01", []models.Event{event}, order)
			if got == nil || got.Phone != "123" {
				t.Fatalf("Select() = %#v, want phone 123", got)
			}
		}
	})

	t.Run("calendar and vcard select one person with multiple emails", func(t *testing.T) {
		ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:duty\nDTSTART;VALUE=DATE:20261101\n" +
			"SUMMARY:Berta 2 ist zuständig\nEND:VEVENT\nEND:VCALENDAR\n"
		vcard := "BEGIN:VCARD\nVERSION:3.0\nN:Zeller;Berta;;;\nTEL:789\n" +
			"EMAIL:invalid\nEMAIL:berta@example.test\nEMAIL:berta.work@example.test\nEND:VCARD\n"
		start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
		events, issues, err := parser.NewCalendarParser().Parse(ical, start, start.AddDate(0, 0, 1))
		if err != nil || len(issues) != 0 {
			t.Fatalf("Parse() issues = %v, error = %v", issues, err)
		}
		contacts, err := parser.ParseVCard(vcard)
		if err != nil || len(contacts) != 1 {
			t.Fatalf("ParseVCard() = %#v, %v", contacts, err)
		}
		p := NewPlanner()
		if got := p.Select("2026-11-01", events, contacts); got == nil || got.Email != "berta@example.test" {
			t.Fatalf("Select() = %#v, want first valid email", got)
		}
		if got := p.Reminder("2026-11-01", events, contacts); got == nil || got.Email != "berta@example.test" {
			t.Fatalf("Reminder() = %#v, want first valid email", got)
		}
		if len(p.Issues(events, contacts)) != 0 || len(p.Gaps("2026-11-01", "2026-11-02", events, contacts)) != 0 {
			t.Fatal("valid title must not produce an issue or gap")
		}
		events[0].Attendees = []string{"berta.work@example.test"}
		if got := p.Select("2026-11-01", events, contacts); got == nil || got.Email != "berta.work@example.test" {
			t.Fatalf("Select() = %#v, want the attendee's email", got)
		}
	})
}

func TestGaps(t *testing.T) {
	p := NewPlanner()

	contact := models.Contact{
		Name:  "John",
		Phone: "123456789",
		Email: "john@example.com",
	}

	events := []models.Event{
		{
			UID:       "1",
			Start:     "2024-01-15",
			End:       "2024-01-16",
			Attendees: []string{"john@example.com"},
			Created:   nil,
		},
		{
			UID:       "2",
			Start:     "2024-01-18",
			End:       "2024-01-19",
			Attendees: []string{"john@example.com"},
			Created:   nil,
		},
	}

	gaps := p.Gaps("2024-01-15", "2024-01-20", events, []models.Contact{contact})

	expectedGaps := []string{"2024-01-16", "2024-01-17", "2024-01-19"}
	if len(gaps) != len(expectedGaps) {
		t.Errorf("Gaps() returned %d gaps, want %d", len(gaps), len(expectedGaps))
	}

	for i, gap := range gaps {
		if i < len(expectedGaps) && gap != expectedGaps[i] {
			t.Errorf("Gap %d = %s, want %s", i, gap, expectedGaps[i])
		}
	}
}

func TestPriorityAndMissingCreationTieBreakAreInputOrderIndependent(t *testing.T) {
	createdOld := "2026-01-01T00:00:00Z"
	createdNew := "2026-01-02T00:00:00Z"
	events := []models.Event{
		{UID: "base", Start: "2026-11-01", End: "2026-11-10", Attendees: []string{"base@example.test"}},
		{UID: "z", Start: "2026-11-03", End: "2026-11-05", Attendees: []string{"z@example.test"}, Created: &createdNew},
		{UID: "a", Start: "2026-11-03", End: "2026-11-05", Attendees: []string{"a@example.test"}, Created: &createdNew},
		{UID: "old", Start: "2026-11-03", End: "2026-11-05", Attendees: []string{"old@example.test"}, Created: &createdOld},
		{UID: "invalid", Start: "2026-11-04", End: "2026-11-05", Attendees: []string{"missing@example.test"}},
	}
	contacts := []models.Contact{
		{Name: "Base", Phone: "111", Email: "base@example.test"},
		{Name: "A", Phone: "222", Email: "a@example.test"},
		{Name: "Z", Phone: "333", Email: "z@example.test"},
		{Name: "Old", Phone: "444", Email: "old@example.test"},
	}
	p := NewPlanner()
	for _, candidates := range [][]models.Event{events, reverseEvents(events)} {
		if got := p.Select("2026-11-04", candidates, contacts); got == nil || got.Name != "A" {
			t.Fatalf("selected = %#v, want A", got)
		}
		if len(p.Issues(candidates, contacts)) == 0 {
			t.Fatal("expected invalid assignment issue")
		}
	}

	a := models.Event{UID: "a", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"a@example.test"}, Created: &createdOld}
	b := models.Event{UID: "b", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"z@example.test"}, Created: &createdNew}
	missing := models.Event{UID: "c", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"base@example.test"}}
	for _, candidates := range [][]models.Event{{a, b}, {b, a}} {
		if got := p.Select("2026-11-01", candidates, contacts); got == nil || got.Name != "Z" {
			t.Fatalf("later created event selected = %#v, want Z", got)
		}
	}
	for _, candidates := range [][]models.Event{{a, b, missing}, {missing, b, a}} {
		if got := p.Select("2026-11-01", candidates, contacts); got == nil || got.Name != "A" {
			t.Fatalf("missing-created UID tie-break selected = %#v, want A", got)
		}
	}
}

func TestUnresolvedContactIssuesIdentifyMissingFields(t *testing.T) {
	p := NewPlanner()
	noEmail, err := parser.ParseVCard("BEGIN:VCARD\nN:Beispiel;Ada;;;\nTEL:123\nEND:VCARD\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		event    models.Event
		contacts []models.Contact
		want     []string
	}{
		{
			name:     "title and missing phone",
			event:    models.Event{UID: "karin", Start: "2026-11-22", End: "2026-11-23", Summary: "Karin ist zuständig"},
			contacts: []models.Contact{{Name: "Karin", LastName: "Beispiel", Email: "karin@example.test"}},
			want:     []string{`"Karin ist zuständig"`, "karin", "2026-11-22", `"Karin Beispiel"`, "Telefonnummer fehlt"},
		},
		{
			name:     "missing email retained from vcard",
			event:    models.Event{UID: "ada", Start: "2026-11-22", End: "2026-11-23", Summary: "Ada ist zuständig"},
			contacts: noEmail,
			want:     []string{`"Ada Beispiel"`, "E-Mail-Adresse fehlt"},
		},
		{
			name:     "attendee identifies missing name and phone",
			event:    models.Event{UID: "guest", Start: "2026-11-22", End: "2026-11-23", Attendees: []string{"guest@example.test"}},
			contacts: []models.Contact{{LastName: "Beispiel", Email: "guest@example.test"}},
			want:     []string{`"Beispiel"`, "guest@example.test", "Vorname fehlt", "Telefonnummer fehlt"},
		},
		{
			name:     "invalid email and missing phone",
			event:    models.Event{UID: "invalid", Start: "2026-11-22", End: "2026-11-23", Summary: "Ada"},
			contacts: []models.Contact{{Name: "Ada", Email: "nicht-gueltig"}},
			want:     []string{"Ada", "Telefonnummer fehlt"},
		},
		{
			name:  "unknown attendee",
			event: models.Event{UID: "unknown", Start: "2026-11-22", End: "2026-11-23", Attendees: []string{"unknown@example.test"}},
			want:  []string{"unknown@example.test", "Adressbuch prüfen"},
		},
		{
			name:     "ambiguous complete contacts",
			event:    models.Event{UID: "ambiguous", Start: "2026-11-22", End: "2026-11-23", Summary: "Ada und Berta"},
			contacts: []models.Contact{{Name: "Ada", Phone: "123", Email: "ada@example.test"}, {Name: "Berta", Phone: "456", Email: "berta@example.test"}},
			want:     []string{"Mehrere vollständige Kontakte", "Zuordnung präzisieren"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Issues([]models.Event{tt.event}, tt.contacts)
			if len(issues) != 1 {
				t.Fatalf("Issues() = %v, want one issue", issues)
			}
			for _, text := range tt.want {
				if !strings.Contains(issues[0], text) {
					t.Errorf("issue %q does not contain %q", issues[0], text)
				}
			}
			if selected := p.Select("2026-11-22", []models.Event{tt.event}, tt.contacts); selected != nil {
				t.Errorf("Select() = %#v, want no assignment", selected)
			}
		})
	}
}

func TestSelectContactWithMissingSurname(t *testing.T) {
	contacts := []models.Contact{{Name: "Clara", Phone: "789", Email: "clara@example.test"}}
	event := models.Event{UID: "duty", Start: "2026-11-22", End: "2026-11-23", Summary: "Clara"}
	if selected := NewPlanner().Select("2026-11-22", []models.Event{event}, contacts); selected == nil || selected.Name != "Clara" {
		t.Fatalf("Select() = %#v, want Clara despite missing surname", selected)
	}
}

func TestUnresolvedDutyDoesNotReportInvalidEmailAddress(t *testing.T) {
	event := models.Event{UID: "duty", Start: "2026-11-22", End: "2026-11-23", Summary: "Ada"}
	contacts := []models.Contact{{Name: "Ada", LastName: "Test", Phone: "123", Email: "bad-address"}}
	issues := NewPlanner().Issues([]models.Event{event}, contacts)
	if len(issues) != 1 || strings.Contains(issues[0], "bad-address") || strings.Contains(issues[0], "E-Mail") {
		t.Fatalf("Issues() = %v, want an unresolved duty without invalid email details", issues)
	}
}

func TestIssuesOnlyReportsEventsWithoutAValidContact(t *testing.T) {
	p := NewPlanner()
	events := []models.Event{
		{UID: "valid", Start: "2026-11-01", End: "2026-11-02", Attendees: []string{"person@example.test"}},
		{UID: "invalid", Start: "2026-11-02", End: "2026-11-03", Attendees: []string{"missing@example.test"}},
	}
	contacts := []models.Contact{{Name: "Person", Phone: "123", Email: "person@example.test"}}

	issues := p.Issues(events[:1], contacts)
	if len(issues) != 0 {
		t.Fatalf("Issues() for valid assignment = %v, want none", issues)
	}
	issues = p.Issues(events, contacts)
	if len(issues) != 1 || !strings.Contains(issues[0], "invalid") {
		t.Fatalf("Issues() = %v, want one issue for invalid event", issues)
	}
}

func reverseEvents(events []models.Event) []models.Event {
	reversed := make([]models.Event, len(events))
	for i := range events {
		reversed[len(events)-1-i] = events[i]
	}
	return reversed
}

func TestReminder(t *testing.T) {
	p := NewPlanner()

	contact := models.Contact{
		Name:  "John",
		Phone: "123456789",
		Email: "john@example.com",
	}

	tests := []struct {
		name     string
		date     string
		events   []models.Event
		expected *models.Contact
	}{
		{
			name: "First day of duty",
			date: "2024-01-15",
			events: []models.Event{
				{
					UID:       "1",
					Start:     "2024-01-15",
					End:       "2024-01-18",
					Attendees: []string{"john@example.com"},
					Created:   nil,
				},
			},
			expected: &contact,
		},
		{
			name: "Continuation of previous duty",
			date: "2024-01-16",
			events: []models.Event{
				{
					UID:       "1",
					Start:     "2024-01-15",
					End:       "2024-01-18",
					Attendees: []string{"john@example.com"},
					Created:   nil,
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Reminder(tt.date, tt.events, []models.Contact{contact})
			if result == nil && tt.expected != nil {
				t.Errorf("Reminder() = nil, want %v", tt.expected)
			}
			if result != nil && tt.expected == nil {
				t.Errorf("Reminder() = %v, want nil", result)
			}
		})
	}
}

func timeZone() *time.Location {
	loc, _ := time.LoadLocation("Europe/Zurich")
	return loc
}
