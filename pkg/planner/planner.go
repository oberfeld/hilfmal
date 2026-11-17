package planner

import (
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"oberfeld.be/hilfmal/pkg/models"
)

// Planner evaluates calendar entries and contacts to determine duty assignments
type Planner struct{}

// NewPlanner creates a new planner
func NewPlanner() *Planner {
	return &Planner{}
}

// ServiceDate returns the calendar day of the current duty
// Before 06:00 Zurich time, the duty belongs to the previous day
func (p *Planner) ServiceDate(now time.Time) (string, error) {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return "", fmt.Errorf("load Swiss timezone: %w", err)
	}
	local := now.In(loc)

	if local.Hour() < 6 {
		local = local.AddDate(0, 0, -1)
	}

	return local.Format("2006-01-02"), nil
}

// Select finds the valid contact with highest priority for a duty day
// Priority: shorter duration, later creation time, then by UID
func (p *Planner) Select(date string, events []models.Event, contacts []models.Contact) *models.Contact {
	var candidates []models.Event

	for _, event := range events {
		if event.Start <= date && date < event.End {
			candidates = append(candidates, event)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Check which duration groups are missing CREATED timestamps
	missingCreated := make(map[int]bool)
	for _, candidate := range candidates {
		startDate, _ := time.Parse("2006-01-02", candidate.Start)
		endDate, _ := time.Parse("2006-01-02", candidate.End)
		days := int(endDate.Sub(startDate).Hours() / 24)
		if candidate.Created == nil {
			missingCreated[days] = true
		}
	}

	// Sort candidates by priority
	sort.Slice(candidates, func(i, j int) bool {
		startI, _ := time.Parse("2006-01-02", candidates[i].Start)
		endI, _ := time.Parse("2006-01-02", candidates[i].End)
		daysI := int(endI.Sub(startI).Hours() / 24)

		startJ, _ := time.Parse("2006-01-02", candidates[j].Start)
		endJ, _ := time.Parse("2006-01-02", candidates[j].End)
		daysJ := int(endJ.Sub(startJ).Hours() / 24)

		// 1. Compare duration
		if daysI != daysJ {
			return daysI < daysJ
		}

		// 2. Compare creation time (if both have it and same duration)
		if !missingCreated[daysI] && candidates[i].Created != nil && candidates[j].Created != nil {
			if *candidates[i].Created != *candidates[j].Created {
				return *candidates[j].Created < *candidates[i].Created // Later creation has priority
			}
		}

		// 3. Compare UIDs
		return candidates[i].UID < candidates[j].UID
	})

	// Find first candidate with valid contact
	for _, event := range candidates {
		if contact := p.resolve(&event, contacts); contact != nil {
			return contact
		}
	}

	return nil
}

// resolve finds a valid contact for an event
func (p *Planner) resolve(event *models.Event, contacts []models.Contact) *models.Contact {
	for _, attendee := range event.Attendees {
		for _, contact := range contacts {
			if strings.EqualFold(contact.Email, attendee) &&
				strings.TrimSpace(contact.Name) != "" &&
				strings.TrimSpace(contact.Phone) != "" &&
				isValidEmail(contact.Email) {
				return &contact
			}
		}
	}
	return nil
}

// Issues reports entries whose attendee cannot be mapped to a complete contact
func (p *Planner) Issues(events []models.Event, contacts []models.Contact) []string {
	var issues []string
	for _, event := range events {
		if p.resolve(&event, contacts) == nil {
			issues = append(issues, "Dienst "+event.UID+" ("+event.Start+"): Kontakt fehlt oder ist nicht eindeutig.")
		}
	}
	return issues
}

// Gaps returns days without a valid contact (from inclusive, to exclusive)
func (p *Planner) Gaps(start, end string, events []models.Event, contacts []models.Contact) []string {
	var gaps []string
	startDate, _ := time.Parse("2006-01-02", start)
	endDate, _ := time.Parse("2006-01-02", end)

	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		if p.Select(day, events, contacts) == nil {
			gaps = append(gaps, day)
		}
	}

	return gaps
}

// Reminder returns the contact on the first day of a continuous duty
// to avoid daily reminders for multi-day duties
func (p *Planner) Reminder(date string, events []models.Event, contacts []models.Contact) *models.Contact {
	current := p.Select(date, events, contacts)
	if current == nil {
		return nil
	}

	prevDate, _ := time.Parse("2006-01-02", date)
	prevDate = prevDate.AddDate(0, 0, -1)
	previous := p.Select(prevDate.Format("2006-01-02"), events, contacts)

	if previous == nil || !strings.EqualFold(current.Email, previous.Email) {
		return current
	}

	return nil
}

// isValidEmail checks if email format is valid
func isValidEmail(email string) bool {
	if strings.TrimSpace(email) != email || strings.ContainsAny(email, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}
