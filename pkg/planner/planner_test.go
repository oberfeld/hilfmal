package planner

import (
	"strings"
	"testing"
	"time"

	"oberfeld.be/hilfmal/pkg/models"
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
