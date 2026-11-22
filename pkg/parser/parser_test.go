package parser

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// @lat: [[tests#Tests#Calendar and contact resolution#Recurrence and all-day parsing]]
func TestRecurrenceExceptionsAndAttendeeStatuses(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:recurring\r\nDTSTART;VALUE=DATE:20261101\r\nDTEND;VALUE=DATE:20261102\r\n" +
		"RRULE:FREQ=DAILY;COUNT=4\r\nEXDATE;VALUE=DATE:20261102\r\n" +
		"CREATED:20260101T000000Z\r\nATTENDEE;PARTSTAT=DECLINED:mailto:a@example.test\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:recurring\r\nRECURRENCE-ID;VALUE=DATE:20261103\r\n" +
		"DTSTART;VALUE=DATE:20261103\r\nDTEND;VALUE=DATE:20261104\r\n" +
		"ATTENDEE;PARTSTAT=ACCEPTED:mailto:b@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-06"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	starts := []string{events[0].Start, events[1].Start, events[2].Start}
	if !reflect.DeepEqual(starts, []string{"2026-11-01", "2026-11-03", "2026-11-04"}) {
		t.Fatalf("occurrence starts = %v", starts)
	}
	if !reflect.DeepEqual(events[0].Attendees, []string{"a@example.test"}) {
		t.Fatalf("master attendee = %v", events[0].Attendees)
	}
	if !reflect.DeepEqual(events[1].Attendees, []string{"b@example.test"}) {
		t.Fatalf("exception attendee = %v", events[1].Attendees)
	}
	if events[1].End != "2026-11-04" {
		t.Fatalf("exception end = %s", events[1].End)
	}
	if events[0].Created == nil || *events[0].Created != "2026-01-01T00:00:00Z" ||
		events[1].Created == nil || *events[1].Created != "2026-01-01T00:00:00Z" {
		t.Fatalf("master/exception creation times = %v/%v", events[0].Created, events[1].Created)
	}
}

func TestOrganizerOnRecurringEvents(t *testing.T) {
	ical := "BEGIN:VCALENDAR\nVERSION:2.0\n" +
		"BEGIN:VEVENT\nUID:series\nDTSTART;VALUE=DATE:20261101\nRRULE:FREQ=DAILY;COUNT=4\n" +
		"SUMMARY:Dienst\\, Ada\nORGANIZER;CN=Organizer:MAILTO:organizer@example.test\nEND:VEVENT\n" +
		"BEGIN:VEVENT\nUID:series\nRECURRENCE-ID;VALUE=DATE:20261102\nDTSTART;VALUE=DATE:20261102\nEND:VEVENT\n" +
		"BEGIN:VEVENT\nUID:series\nRECURRENCE-ID;VALUE=DATE:20261103\nDTSTART;VALUE=DATE:20261103\n" +
		"SUMMARY:Anderer Dienst\nORGANIZER:mailto:replacement@example.test\nEND:VEVENT\n" +
		"BEGIN:VEVENT\nUID:series\nRECURRENCE-ID;VALUE=DATE:20261104\nDTSTART;VALUE=DATE:20261104\nSUMMARY:\nEND:VEVENT\nEND:VCALENDAR\n"
	events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 || len(events) != 4 {
		t.Fatalf("events = %#v, issues = %#v", events, issues)
	}
	for i, want := range []string{"organizer@example.test", "organizer@example.test", "replacement@example.test", "organizer@example.test"} {
		if events[i].Organizer != want {
			t.Errorf("event %d organizer = %q, want %q", i, events[i].Organizer, want)
		}
	}
	for i, want := range []string{"Dienst, Ada", "Dienst, Ada", "Anderer Dienst", ""} {
		if events[i].Summary != want {
			t.Errorf("event %d summary = %q, want %q", i, events[i].Summary, want)
		}
	}
}

func TestInvalidEntriesAreReportedWithoutDiscardingValidOnes(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:bad\r\nDTSTART:20261101T080000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:good\r\nDTSTART;VALUE=DATE:20261101\r\nATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-03"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UID != "good" {
		t.Fatalf("events = %#v, want only valid event", events)
	}
	if len(issues) == 0 {
		t.Fatal("expected invalid event issue")
	}
}

// @lat: [[tests#Tests#Calendar and contact resolution#Identifying invalid start dates]]
func TestInvalidStartIssueIdentifiesEvent(t *testing.T) {
	tests := []struct {
		name       string
		properties string
		want       string
	}{
		{
			name:       "UTC timed event",
			properties: "SUMMARY:Dienst\\, Beispiel\nDTSTART:20261123T100000Z\n",
			want:       `Dienst "Dienst, Beispiel" (UID: invalid, Start: 23.11.2026 10:00:00 UTC) benötigt ein ganztägiges Startdatum.`,
		},
		{
			name:       "time with timezone",
			properties: "SUMMARY:Testdienst\nDTSTART;TZID=Europe/Zurich:20261123T110000\n",
			want:       `Dienst "Testdienst" (UID: invalid, Start: 23.11.2026 11:00:00 ("Europe/Zurich")) benötigt ein ganztägiges Startdatum.`,
		},
		{
			name:       "time without timezone",
			properties: "SUMMARY:Testdienst\nDTSTART:20261123T110000\n",
			want:       `Dienst "Testdienst" (UID: invalid, Start: 23.11.2026 11:00:00 (Zeitzone nicht angegeben)) benötigt ein ganztägiges Startdatum.`,
		},
		{
			name:       "missing start and title",
			properties: "",
			want:       `Dienst "(ohne Titel)" (UID: invalid, Start: fehlt) benötigt ein ganztägiges Startdatum.`,
		},
		{
			name:       "malformed start",
			properties: "SUMMARY:Testdienst\nDTSTART:kein Datum\n",
			want:       `Dienst "Testdienst" (UID: invalid, Start: "kein Datum") benötigt ein ganztägiges Startdatum.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:invalid\n" + tt.properties + "END:VEVENT\nEND:VCALENDAR\n"
			events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-12-01"))
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 0 || len(issues) != 1 || issues[0] != tt.want {
				t.Fatalf("events = %#v, issues = %#v; want issue %q", events, issues, tt.want)
			}
		})
	}
}

func TestCancelledOccurrenceIsNotAssigned(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:recurring\r\nDTSTART;VALUE=DATE:20261101\r\nRRULE:FREQ=DAILY;COUNT=2\r\n" +
		"ATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:recurring\r\nRECURRENCE-ID;VALUE=DATE:20261102\r\n" +
		"DTSTART;VALUE=DATE:20261102\r\nSTATUS:CANCELLED\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-04"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Start != "2026-11-01" {
		t.Fatalf("events = %#v, want only first occurrence", events)
	}
}

func TestAllDayDurationCoversAllDays(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:long\r\nDTSTART;VALUE=DATE:20261101\r\nDURATION:P3D\r\n" +
		"ATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 || len(events) != 1 || events[0].End != "2026-11-04" {
		t.Fatalf("events=%#v issues=%v", events, issues)
	}
}

func TestMovedExceptionAndFoldedAttendee(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:series\r\nDTSTART;VALUE=DATE:20261101\r\nRRULE:FREQ=DAILY;COUNT=3\r\n" +
		"ATTENDEE:mailto:long-address-that-is-folded@example.\r\n test\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:series\r\nRECURRENCE-ID;VALUE=DATE:20261102\r\n" +
		"DTSTART;VALUE=DATE:20261110\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-10"), date("2026-11-11"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Start != "2026-11-10" {
		t.Fatalf("moved occurrence = %#v", events)
	}
	if !reflect.DeepEqual(events[0].Attendees, []string{"long-address-that-is-folded@example.test"}) {
		t.Fatalf("attendees = %v", events[0].Attendees)
	}
}

func TestMalformedCalendarReturnsError(t *testing.T) {
	_, _, err := NewCalendarParser().Parse("BEGIN:VEVENT\nEND:VEVENT", date("2026-11-01"), date("2026-11-02"))
	if err == nil {
		t.Fatal("expected malformed calendar error")
	}
}

func TestParseVCardUsesGivenNameAndEmails(t *testing.T) {
	vcard := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Ada Lovelace\r\nN:Lovelace;Ada;;;\r\n" +
		"TEL:+41 12\r\nEMAIL:ada@example.test\r\nEMAIL:ada.work@example.test\r\nEND:VCARD\r\n"
	contacts, err := ParseVCard(vcard)
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0].Name != "Ada" || contacts[0].LastName != "Lovelace" ||
		contacts[0].Phone != "+41 12" || contacts[0].Email != "ada@example.test" ||
		!reflect.DeepEqual(contacts[0].AdditionalEmails, []string{"ada.work@example.test"}) {
		t.Fatalf("contacts = %#v", contacts)
	}
}

func TestParseVCardEscapedNameSeparator(t *testing.T) {
	vcard := "BEGIN:VCARD\nVERSION:3.0\nN:Van\\;Buren;Ada;;;\nEMAIL:ada@example.test\nEND:VCARD\n"
	contacts, err := ParseVCard(vcard)
	if err != nil || len(contacts) != 1 || contacts[0].Name != "Ada" || contacts[0].LastName != "Van;Buren" {
		t.Fatalf("ParseVCard() = %#v, %v", contacts, err)
	}
}

func TestParseVCardPreservesContactWithoutEmail(t *testing.T) {
	vcard := "BEGIN:VCARD\nVERSION:3.0\nUID:ada\nFN:Ada Beispiel\nN:Beispiel;Ada;;;\nTEL:123\nEND:VCARD\n"
	contacts, err := ParseVCard(vcard)
	if err != nil || len(contacts) != 1 || contacts[0].Name != "Ada" || contacts[0].LastName != "Beispiel" || contacts[0].DisplayName != "Ada Beispiel" || contacts[0].UID != "ada" || contacts[0].Phone != "123" || contacts[0].Email != "" {
		t.Fatalf("ParseVCard() = %#v, %v", contacts, err)
	}
}

func TestParseVCardRejectsMalformedDataAndPreservesInvalidEmailForValidation(t *testing.T) {
	if _, err := ParseVCard("EMAIL:ada@example.test"); err == nil {
		t.Fatal("expected missing VCARD wrapper to fail")
	}
	emails, err := ParseVCardEmails("BEGIN:VCARD\nEMAIL:not-an-email\nEND:VCARD")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(emails, []string{"not-an-email"}) {
		t.Fatalf("emails = %v", emails)
	}
}

func TestParseCalendarSortsByStartThenUID(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:z\r\nDTSTART;VALUE=DATE:20261102\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:b\r\nDTSTART;VALUE=DATE:20261101\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:a\r\nDTSTART;VALUE=DATE:20261101\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-03"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].UID != "a" || events[1].UID != "b" || events[2].UID != "z" {
		t.Fatalf("sorted events = %#v, want a, b, z", events)
	}
}

func TestParseCalendarHandlesNestedComponents(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nBEGIN:VALARM\r\nUID:ignored\r\nDTSTART;VALUE=DATE:20261102\r\n" +
		"END:VALARM\r\nUID:with-alarm\r\nDTSTART;VALUE=DATE:20261101\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-02"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UID != "with-alarm" {
		t.Fatalf("events = %#v, want event containing nested alarm", events)
	}
}

func TestParseCalendarIgnoresTimezoneComponent(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VTIMEZONE\r\nTZID:Europe/Zurich\r\nDTSTART:20260329T020000\r\nEND:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\nUID:service\r\nDTSTART;VALUE=DATE:20261101\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-02"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UID != "service" {
		t.Fatalf("events = %#v, want the service event after timezone metadata", events)
	}
}

func TestParseCalendarIncludesRDateOccurrences(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:rdate\r\nDTSTART;VALUE=DATE:20261101\r\n" +
		"RDATE;VALUE=DATE:20261103\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	events, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Start != "2026-11-01" || events[1].Start != "2026-11-03" {
		t.Fatalf("RDATE occurrences = %#v", events)
	}
}

func TestParseReportsAndKeepsStandaloneException(t *testing.T) {
	ical := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:orphan\r\nRECURRENCE-ID;VALUE=DATE:20261101\r\n" +
		"DTSTART;VALUE=DATE:20261101\r\nATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, issues, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-02"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UID != "orphan" {
		t.Fatalf("standalone exception events = %#v", events)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "Ausnahme ohne Serieneintrag") {
		t.Fatalf("standalone exception issues = %v", issues)
	}
}

func TestMalformedLeadingFoldedLineReturnsError(t *testing.T) {
	ical := " continuation\r\nBEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n"
	if _, _, err := NewCalendarParser().Parse(ical, date("2026-11-01"), date("2026-11-02")); err == nil {
		t.Fatal("expected leading folded line to be rejected")
	}
}

func TestParsePropertyRejectsEmptyName(t *testing.T) {
	if _, ok := parseProperty(":value"); ok {
		t.Fatal("parseProperty() accepted an empty property name")
	}
}

func TestAllDayDurationRejectsZeroAndSupportsWeeks(t *testing.T) {
	start := date("2026-11-01")
	if _, err := allDayDurationEnd(start, "P0D"); err == nil {
		t.Fatal("allDayDurationEnd() accepted a zero-day duration")
	}
	if got, err := allDayDurationEnd(start, "P1W"); err != nil || !got.Equal(date("2026-11-08")) {
		t.Fatalf("allDayDurationEnd(P1W) = %s, %v", got, err)
	}
}

func TestParseCreatedSupportsBothFormatsAndRejectsInvalidValue(t *testing.T) {
	tests := []struct {
		value string
		want  string
		ok    bool
	}{
		{value: "2026-01-01T00:00:00Z", want: "2026-01-01T00:00:00Z", ok: true},
		{value: "20260101T000000Z", want: "2026-01-01T00:00:00Z", ok: true},
		{value: "invalid", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := parseCreated(tt.value)
			if (err == nil) != tt.ok || got != tt.want {
				t.Fatalf("parseCreated(%q) = %q, %v; want %q, success %t", tt.value, got, err, tt.want, tt.ok)
			}
		})
	}
}

func TestParseVCardIgnoresMalformedNameWithoutPanicking(t *testing.T) {
	vcard := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Mononym\r\nEMAIL:person@example.test\r\nEND:VCARD\r\n"
	contacts, err := ParseVCard(vcard)
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0].Name != "" {
		t.Fatalf("contacts = %#v, want one contact with no parsed given name", contacts)
	}
}

func TestWithoutMailtoHandlesPrefixBoundariesAndCase(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "mailto:", want: ""},
		{value: "MAILTO:ada@example.test", want: "ada@example.test"},
		{value: "not-an-email", want: "not-an-email"},
		{value: "short", want: "short"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := withoutMailto(tt.value); got != tt.want {
				t.Fatalf("withoutMailto(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func date(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}
