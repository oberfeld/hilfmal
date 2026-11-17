package parser

import (
	"reflect"
	"testing"
	"time"
)

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
	if len(contacts) != 2 || contacts[0].Name != "Ada" || contacts[0].Phone != "+41 12" ||
		contacts[0].Email != "ada@example.test" || contacts[1].Email != "ada.work@example.test" {
		t.Fatalf("contacts = %#v", contacts)
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

func date(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}
