package planner

import (
	"fmt"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode"

	contactcheck "oberfeld.be/hilfmal/pkg/contacts"
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
		if contact, _ := p.resolve(&event, contacts); contact != nil {
			return contact
		}
	}

	return nil
}

// resolve prefers other attendees, then the event title, then the organizer.
func (p *Planner) resolve(event *models.Event, contacts []models.Contact) (*models.Contact, bool) {
	organizer, organizerIndex := resolveContact(event.Organizer, contacts)
	for _, attendee := range event.Attendees {
		contact, index := resolveContact(attendee, contacts)
		if contact != nil && index != organizerIndex {
			return contact, false
		}
	}
	if contact, ambiguous := resolveSummary(event.Summary, contacts); contact != nil || ambiguous {
		return contact, ambiguous
	}
	return organizer, false
}

func resolveContact(email string, contacts []models.Contact) (*models.Contact, int) {
	for i, contact := range contacts {
		if strings.TrimSpace(contact.Name) == "" || strings.TrimSpace(contact.Phone) == "" {
			continue
		}
		for _, address := range append([]string{contact.Email}, contact.AdditionalEmails...) {
			if strings.EqualFold(address, email) && contactcheck.ValidEmail(address) {
				contact.Email = address
				return &contact, i
			}
		}
	}
	return nil, -1
}

func resolveSummary(summary string, contacts []models.Contact) (*models.Contact, bool) {
	var candidates []models.Contact
	for _, contact := range contacts {
		if strings.TrimSpace(contact.Name) == "" || strings.TrimSpace(contact.Phone) == "" ||
			!containsName(summary, contact.Name) {
			continue
		}
		for _, address := range append([]string{contact.Email}, contact.AdditionalEmails...) {
			if contactcheck.ValidEmail(address) {
				contact.Email = address
				candidates = append(candidates, contact)
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}
	if len(candidates) == 1 {
		return &candidates[0], false
	}

	var matches []models.Contact
	for i := range candidates {
		if !containsDistinctNames(summary, candidates[i].Name, candidates[i].LastName) {
			continue
		}
		matches = append(matches, candidates[i])
	}
	if len(matches) == 0 {
		for _, candidate := range candidates[1:] {
			if !strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(candidates[0].Name)) {
				return nil, true
			}
		}
		matches = candidates
	} else {
		for _, match := range matches[1:] {
			if !strings.EqualFold(strings.TrimSpace(match.Name), strings.TrimSpace(matches[0].Name)) ||
				!strings.EqualFold(strings.TrimSpace(match.LastName), strings.TrimSpace(matches[0].LastName)) {
				return nil, true
			}
		}
	}
	if len(matches) == 1 {
		return &matches[0], false
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		for _, pair := range [][2]string{
			{strings.ToLower(a.Email), strings.ToLower(b.Email)},
			{a.Email, b.Email},
			{a.Phone, b.Phone},
			{a.Name, b.Name},
			{a.LastName, b.LastName},
		} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
	return &matches[0], true
}

func containsName(text, name string) bool {
	return len(nameRanges(text, name)) > 0
}

func containsDistinctNames(text, firstName, lastName string) bool {
	for _, first := range nameRanges(text, firstName) {
		for _, last := range nameRanges(text, lastName) {
			if first[1] <= last[0] || last[1] <= first[0] {
				return true
			}
		}
	}
	return false
}

func nameRanges(text, name string) [][2]int {
	textRunes := []rune(text)
	nameRunes := []rune(strings.TrimSpace(name))
	if len(nameRunes) == 0 {
		return nil
	}
	isWord := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
	}
	var matches [][2]int
	for start := 0; start+len(nameRunes) <= len(textRunes); start++ {
		end := start + len(nameRunes)
		if (start == 0 || !isWord(textRunes[start-1])) &&
			(end == len(textRunes) || !isWord(textRunes[end])) &&
			strings.EqualFold(string(textRunes[start:end]), string(nameRunes)) {
			matches = append(matches, [2]int{start, end})
		}
	}
	return matches
}

// Issues reports unresolved or ambiguous contact assignments.
func (p *Planner) Issues(events []models.Event, contacts []models.Contact) []string {
	var issues []string
	for _, event := range events {
		contact, ambiguous := p.resolve(&event, contacts)
		if contact == nil {
			issues = append(issues, unresolvedContactIssue(event, contacts, ambiguous))
		} else if ambiguous {
			issues = append(issues, fmt.Sprintf("Dienst %s (%s): Mehrere Kontakte passen zum Titel; %s (Telefon: %s) wurde ausgewählt. Bitte Zuordnung prüfen.",
				event.UID, event.Start, contact.Email, contact.Phone))
		}
	}
	return issues
}

func unresolvedContactIssue(event models.Event, contacts []models.Contact, ambiguous bool) string {
	title := event.Summary
	if strings.TrimSpace(title) == "" {
		title = "(ohne Titel)"
	}
	prefix := fmt.Sprintf("Dienst %q (UID: %s, Datum: %s): ", title, event.UID, event.Start)
	if ambiguous {
		return prefix + "Mehrere vollständige Kontakte passen zum Titel; bitte die Zuordnung präzisieren."
	}

	var candidates []string
	for _, contact := range contacts {
		matches := containsName(event.Summary, contact.Name)
		for _, address := range append([]string{contact.Email}, contact.AdditionalEmails...) {
			if address == "" {
				continue
			}
			if strings.EqualFold(address, event.Organizer) {
				matches = true
			}
			for _, attendee := range event.Attendees {
				if strings.EqualFold(address, attendee) {
					matches = true
				}
			}
		}
		if !matches {
			continue
		}

		if issue := (contactcheck.Contacts{}).Issue(contact); issue != "" {
			candidates = append(candidates, issue)
		}
	}
	if len(candidates) > 0 {
		sort.Strings(candidates)
		return prefix + "Mögliche unvollständige Kontakte: " + strings.Join(candidates, "; ") + "."
	}
	var addresses []string
	for _, attendee := range event.Attendees {
		if contactcheck.ValidEmail(attendee) {
			addresses = append(addresses, fmt.Sprintf("Einladung %q", attendee))
		}
	}
	if contactcheck.ValidEmail(event.Organizer) {
		addresses = append(addresses, fmt.Sprintf("Organisator %q", event.Organizer))
	}
	if len(addresses) > 0 {
		return prefix + "Kein passender vollständiger Kontakt gefunden; " + strings.Join(addresses, ", ") + " und Adressbuch prüfen."
	}
	return prefix + "Kein passender vollständiger Kontakt gefunden; Titel und Adressbuch prüfen."
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
