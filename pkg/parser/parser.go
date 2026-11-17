package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
	"oberfeld.be/hilfmal/pkg/models"
)

type property struct {
	name   string
	params map[string]string
	value  string
}

type calendarEvent struct {
	uid          string
	start        time.Time
	end          time.Time
	attendees    []string
	created      *string
	status       string
	rules        []string
	rdates       []time.Time
	exdates      []time.Time
	recurrenceID *time.Time
}

// CalendarParser parses all-day iCalendar events and expands recurring events.
type CalendarParser struct {
	issues []string
}

func NewCalendarParser() *CalendarParser {
	return &CalendarParser{}
}

func (p *CalendarParser) Issues() []string {
	return append([]string(nil), p.issues...)
}

// Parse returns events overlapping [from, to), parser issues, and a fatal
// error for a malformed calendar document.
func (p *CalendarParser) Parse(ical string, from, to time.Time) ([]models.Event, []string, error) {
	p.issues = nil
	if !from.Before(to) {
		return nil, nil, fmt.Errorf("calendar range must have a positive duration")
	}

	rawEvents, err := readCalendar(ical)
	if err != nil {
		return nil, nil, err
	}

	masters := make(map[string]*calendarEvent)
	overrides := make(map[string]map[string]*calendarEvent)
	for _, raw := range rawEvents {
		event, err := parseCalendarEvent(raw, &p.issues)
		if err != nil {
			p.issues = append(p.issues, err.Error())
			continue
		}
		if event == nil {
			continue
		}
		if event.recurrenceID == nil {
			masters[event.uid] = event
			continue
		}
		if overrides[event.uid] == nil {
			overrides[event.uid] = make(map[string]*calendarEvent)
		}
		overrides[event.uid][event.recurrenceID.Format("2006-01-02")] = event
	}

	fromDate := dateOnly(from)
	toDate := dateOnly(to)
	result := make([]models.Event, 0)
	for uid, master := range masters {
		if master.status == "CANCELLED" {
			continue
		}
		instances, err := expandEvent(master, fromDate, toDate)
		if err != nil {
			p.issues = append(p.issues, fmt.Sprintf("Kalendereintrag %s hat eine ungültige Wiederholung: %v", uid, err))
			continue
		}
		seenOverrides := make(map[string]bool)
		for _, start := range instances {
			originalDate := start.Format("2006-01-02")
			instance := master
			if override := overrides[uid][originalDate]; override != nil {
				instance = mergeOverride(master, override)
				seenOverrides[originalDate] = true
			} else {
				instance = copyEventWithDates(master, start, start.Add(master.end.Sub(master.start)))
			}
			if instance.status == "CANCELLED" || !overlaps(instance.start, instance.end, fromDate, toDate) {
				continue
			}
			result = append(result, toModelEvent(uid, instance))
		}

		// A moved occurrence can land in the requested range even when its
		// original recurrence date does not.
		for recurrenceDate, override := range overrides[uid] {
			if seenOverrides[recurrenceDate] || override.status == "CANCELLED" {
				continue
			}
			instance := mergeOverride(master, override)
			if overlaps(instance.start, instance.end, fromDate, toDate) {
				result = append(result, toModelEvent(uid, instance))
			}
		}
	}

	for uid, events := range overrides {
		if masters[uid] != nil {
			continue
		}
		for _, event := range events {
			p.issues = append(p.issues, fmt.Sprintf("Kalendereintrag %s hat eine Ausnahme ohne Serieneintrag.", uid))
			if event.status != "CANCELLED" && overlaps(event.start, event.end, fromDate, toDate) {
				result = append(result, toModelEvent(uid, event))
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Start != result[j].Start {
			return result[i].Start < result[j].Start
		}
		return result[i].UID < result[j].UID
	})
	return result, uniqueStrings(p.issues), nil
}

func readCalendar(input string) ([][]property, error) {
	lines := unfoldLines(input)
	inCalendar := false
	calendarClosed := false
	var current []property
	eventDepth := 0
	var events [][]property

	for _, line := range lines {
		if line == "" {
			continue
		}
		prop, ok := parseProperty(line)
		if !ok {
			return nil, fmt.Errorf("ungültige iCalendar-Zeile")
		}
		switch prop.name {
		case "BEGIN":
			switch strings.ToUpper(prop.value) {
			case "VCALENDAR":
				if inCalendar || calendarClosed {
					return nil, fmt.Errorf("ungültiger VCALENDAR-Aufbau")
				}
				inCalendar = true
			case "VEVENT":
				if !inCalendar || eventDepth != 0 {
					return nil, fmt.Errorf("ungültiger VEVENT-Aufbau")
				}
				current = nil
				eventDepth = 1
			default:
				if eventDepth > 0 {
					eventDepth++
				}
			}
		case "END":
			switch strings.ToUpper(prop.value) {
			case "VCALENDAR":
				if !inCalendar || eventDepth != 0 {
					return nil, fmt.Errorf("ungültiger VCALENDAR-Abschluss")
				}
				inCalendar = false
				calendarClosed = true
			case "VEVENT":
				if eventDepth != 1 {
					return nil, fmt.Errorf("ungültiger VEVENT-Abschluss")
				}
				events = append(events, current)
				current = nil
				eventDepth = 0
			default:
				if eventDepth > 1 {
					eventDepth--
				}
			}
		default:
			if inCalendar && eventDepth == 1 {
				current = append(current, prop)
			}
		}
	}
	if inCalendar || !calendarClosed || eventDepth != 0 {
		return nil, fmt.Errorf("unvollständiger VCALENDAR")
	}
	return events, nil
}

func unfoldLines(input string) []string {
	physical := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	logical := make([]string, 0, len(physical))
	for _, line := range physical {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(logical) > 0 {
			logical[len(logical)-1] += line[1:]
			continue
		}
		logical = append(logical, line)
	}
	return logical
}

func parseProperty(line string) (property, bool) {
	separator := strings.IndexByte(line, ':')
	if separator <= 0 {
		return property{}, false
	}
	parts := strings.Split(line[:separator], ";")
	result := property{name: strings.ToUpper(parts[0]), params: make(map[string]string), value: line[separator+1:]}
	for _, parameter := range parts[1:] {
		key, value, found := strings.Cut(parameter, "=")
		if found {
			result.params[strings.ToUpper(key)] = strings.Trim(value, `"`)
		}
	}
	return result, true
}

func parseCalendarEvent(properties []property, issues *[]string) (*calendarEvent, error) {
	values := make(map[string][]property)
	for _, prop := range properties {
		values[prop.name] = append(values[prop.name], prop)
	}
	uid := firstValue(values["UID"])
	if uid == "" {
		return nil, fmt.Errorf("Kalendereintrag benötigt eine UID.")
	}
	startProp, hasStart := first(values["DTSTART"])
	if !hasStart {
		return nil, fmt.Errorf("Dienst %s benötigt ein ganztägiges Startdatum.", uid)
	}
	start, err := eventDate(startProp)
	if err != nil {
		return nil, fmt.Errorf("Dienst %s benötigt ein ganztägiges Startdatum.", uid)
	}

	end := start.AddDate(0, 0, 1)
	if prop, ok := first(values["DTEND"]); ok {
		end, err = eventDate(prop)
		if err != nil {
			return nil, fmt.Errorf("Dienst %s hat ein ungültiges Enddatum.", uid)
		}
	} else if prop, ok := first(values["DURATION"]); ok {
		end, err = allDayDurationEnd(start, prop.value)
		if err != nil {
			return nil, fmt.Errorf("Dienst %s hat eine ungültige Dauer.", uid)
		}
	}
	if !end.After(start) {
		return nil, fmt.Errorf("Dienst %s hat einen ungültigen Zeitraum.", uid)
	}

	event := &calendarEvent{uid: uid, start: start, end: end}
	if prop, ok := first(values["CREATED"]); ok {
		if created, err := parseCreated(prop.value); err == nil {
			event.created = &created
		}
	}
	if prop, ok := first(values["STATUS"]); ok {
		event.status = strings.ToUpper(prop.value)
	}
	for _, prop := range values["ATTENDEE"] {
		email := prop.value
		if len(email) >= len("mailto:") && strings.EqualFold(email[:len("mailto:")], "mailto:") {
			email = email[len("mailto:"):]
		}
		event.attendees = append(event.attendees, email)
	}
	for _, prop := range values["RRULE"] {
		event.rules = append(event.rules, prop.value)
	}
	for _, name := range []string{"RDATE", "EXDATE"} {
		for _, prop := range values[name] {
			for _, value := range strings.Split(prop.value, ",") {
				date, err := eventDate(property{name: prop.name, params: prop.params, value: value})
				if err != nil {
					*issues = append(*issues, fmt.Sprintf("Dienst %s enthält ein ungültiges %s-Datum.", uid, name))
					continue
				}
				if name == "RDATE" {
					event.rdates = append(event.rdates, date)
				} else {
					event.exdates = append(event.exdates, date)
				}
			}
		}
	}
	if prop, ok := first(values["RECURRENCE-ID"]); ok {
		recurrenceID, err := eventDate(prop)
		if err != nil {
			return nil, fmt.Errorf("Dienst %s hat eine ungültige Wiederholungsausnahme.", uid)
		}
		event.recurrenceID = &recurrenceID
	}
	return event, nil
}

func expandEvent(event *calendarEvent, from, to time.Time) ([]time.Time, error) {
	duration := event.end.Sub(event.start)
	lower := from.Add(-duration)
	if len(event.rules) == 0 {
		occurrences := []time.Time{event.start}
		occurrences = append(occurrences, event.rdates...)
		return filterRecurrenceDates(occurrences, event.exdates, lower, to), nil
	}

	lines := []string{"DTSTART:" + event.start.Format("20060102T000000Z")}
	for _, rule := range event.rules {
		lines = append(lines, "RRULE:"+rule)
	}
	for _, date := range event.rdates {
		lines = append(lines, "RDATE:"+date.Format("20060102T000000Z"))
	}
	for _, date := range event.exdates {
		lines = append(lines, "EXDATE:"+date.Format("20060102T000000Z"))
	}
	set, err := rrule.StrToRRuleSet(strings.Join(lines, "\n"))
	if err != nil {
		return nil, err
	}
	return set.Between(lower, to, true), nil
}

func filterRecurrenceDates(dates, exclusions []time.Time, from, to time.Time) []time.Time {
	excluded := make(map[string]bool, len(exclusions))
	for _, date := range exclusions {
		excluded[date.Format("2006-01-02")] = true
	}
	unique := make(map[string]time.Time)
	for _, date := range dates {
		key := date.Format("2006-01-02")
		if !excluded[key] && !date.Before(from) && date.Before(to) {
			unique[key] = date
		}
	}
	result := make([]time.Time, 0, len(unique))
	for _, date := range unique {
		result = append(result, date)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Before(result[j]) })
	return result
}

func mergeOverride(master, override *calendarEvent) *calendarEvent {
	copy := *override
	if len(copy.attendees) == 0 {
		copy.attendees = append([]string(nil), master.attendees...)
	}
	if copy.created == nil {
		copy.created = master.created
	}
	return &copy
}

func copyEventWithDates(event *calendarEvent, start, end time.Time) *calendarEvent {
	copy := *event
	copy.start = start
	copy.end = end
	copy.attendees = append([]string(nil), event.attendees...)
	return &copy
}

func toModelEvent(uid string, event *calendarEvent) models.Event {
	return models.Event{
		UID:       uid,
		Start:     event.start.Format("2006-01-02"),
		End:       event.end.Format("2006-01-02"),
		Attendees: append([]string(nil), event.attendees...),
		Created:   event.created,
	}
}

func overlaps(start, end, from, to time.Time) bool {
	return start.Before(to) && end.After(from)
}

func dateOnly(date time.Time) time.Time {
	year, month, day := date.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func eventDate(prop property) (time.Time, error) {
	value := prop.value
	if len(value) != 8 {
		return time.Time{}, fmt.Errorf("not a date value")
	}
	parsed, err := time.ParseInLocation("20060102", value, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func allDayDurationEnd(start time.Time, value string) (time.Time, error) {
	if len(value) < 3 || value[0] != 'P' {
		return time.Time{}, fmt.Errorf("invalid all-day duration")
	}
	count, err := strconv.Atoi(value[1 : len(value)-1])
	if err != nil || count <= 0 {
		return time.Time{}, fmt.Errorf("invalid all-day duration")
	}
	switch value[len(value)-1] {
	case 'D':
		return start.AddDate(0, 0, count), nil
	case 'W':
		return start.AddDate(0, 0, count*7), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported all-day duration")
	}
}

func parseCreated(value string) (string, error) {
	for _, layout := range []string{time.RFC3339, "20060102T150405Z"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("invalid CREATED value")
}

func first(properties []property) (property, bool) {
	if len(properties) == 0 {
		return property{}, false
	}
	return properties[0], true
}

func firstValue(properties []property) string {
	prop, ok := first(properties)
	if !ok {
		return ""
	}
	return prop.value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

// ParseVCard extracts the first name, telephone number, and one contact per
// email property.
func ParseVCard(vcard string) ([]models.Contact, error) {
	var firstName, phone string
	var emails []string
	properties, err := readVCard(vcard)
	if err != nil {
		return nil, err
	}
	for _, prop := range properties {
		switch prop.name {
		case "N":
			parts := strings.Split(prop.value, ";")
			if len(parts) > 1 {
				firstName = unescapeText(parts[1])
			}
		case "TEL":
			if phone == "" {
				phone = prop.value
			}
		case "EMAIL":
			emails = append(emails, withoutMailto(strings.TrimSpace(prop.value)))
		}
	}
	contacts := make([]models.Contact, 0, len(emails))
	for _, email := range emails {
		contacts = append(contacts, models.Contact{Name: firstName, Phone: phone, Email: email})
	}
	return contacts, nil
}

func ParseVCardEmails(vcard string) ([]string, error) {
	var emails []string
	properties, err := readVCard(vcard)
	if err != nil {
		return nil, err
	}
	for _, prop := range properties {
		if prop.name == "EMAIL" {
			emails = append(emails, withoutMailto(strings.TrimSpace(prop.value)))
		}
	}
	return emails, nil
}

func readVCard(input string) ([]property, error) {
	lines := unfoldLines(input)
	var properties []property
	inCard := false
	closed := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		prop, ok := parseProperty(line)
		if !ok {
			return nil, fmt.Errorf("ungültige vCard-Zeile")
		}
		if prop.name == "BEGIN" && strings.EqualFold(prop.value, "VCARD") && !inCard && !closed {
			inCard = true
			continue
		}
		if prop.name == "END" && strings.EqualFold(prop.value, "VCARD") && inCard {
			inCard = false
			closed = true
			continue
		}
		if inCard {
			properties = append(properties, prop)
		} else if !closed {
			return nil, fmt.Errorf("ungültiger vCard-Aufbau")
		} else {
			return nil, fmt.Errorf("zusätzlicher Inhalt nach vCard")
		}
	}
	if inCard || !closed {
		return nil, fmt.Errorf("unvollständige vCard")
	}
	return properties, nil
}

func withoutMailto(value string) string {
	if len(value) >= len("mailto:") && strings.EqualFold(value[:len("mailto:")], "mailto:") {
		return value[len("mailto:"):]
	}
	return value
}

func unescapeText(value string) string {
	value = strings.ReplaceAll(value, `\n`, "\n")
	value = strings.ReplaceAll(value, `\,`, ",")
	value = strings.ReplaceAll(value, `\;`, ";")
	return strings.ReplaceAll(value, `\\`, `\`)
}
