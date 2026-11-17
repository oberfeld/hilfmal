package main

import (
	"net/http"
	"strings"
	"time"
)

type davPaths struct {
	calendarBase string
	calendar     string
	addressBase  string
	contacts     string
	admins       string
}

type testContact struct {
	uid, name, phone, email string
}

type testEvent struct {
	uid, email string
	start, end time.Time
}

func setupNextcloud() (davPaths, error) {
	addressBase := "/remote.php/dav/addressbooks/users/" + davUser + "/"
	calendarBase := "/remote.php/dav/calendars/" + davUser + "/"
	for _, name := range []string{"mitwirkende", "administration"} {
		if err := addAddressBook(addressBase, name); err != nil {
			return davPaths{}, err
		}
	}
	if err := addCalendar(calendarBase, "dienstplan"); err != nil {
		return davPaths{}, err
	}

	contactsPath := addressBase + "mitwirkende/"
	for _, contact := range []testContact{
		{uid: "ada", name: "Ada", phone: "111", email: "ada@example.test"},
		{uid: "bea", name: "Bea", phone: "222", email: "bea@example.test"},
		{uid: "cara", name: "Cara", phone: "333", email: "cara@example.test"},
	} {
		if err := addCard(contactsPath, contact); err != nil {
			return davPaths{}, err
		}
	}
	adminsPath := addressBase + "administration/"
	admin := testContact{uid: "admin", name: "Admin", phone: "444", email: "admin@example.test"}
	if err := addCard(adminsPath, admin); err != nil {
		return davPaths{}, err
	}

	return davPaths{
		calendarBase: calendarBase,
		calendar:     calendarBase + "dienstplan/",
		addressBase:  addressBase,
		contacts:     contactsPath,
		admins:       adminsPath,
	}, nil
}

func integrationDate() (time.Time, time.Time, error) {
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	now := time.Now().In(location)
	today, err := serviceDate(now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	fixedTime := time.Date(today.Year(), today.Month(), today.Day(), 6, 0, 0, 0, location)
	return today, fixedTime, nil
}

func serviceDate(now time.Time) (time.Time, error) {
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return time.Time{}, err
	}
	now = now.In(location)
	if now.Hour() < 6 {
		now = now.AddDate(0, 0, -1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location), nil
}

func seedCalendar(calendarPath string, today time.Time) error {
	events := []testEvent{
		{
			uid: "base", email: "ada@example.test",
			start: today.AddDate(0, 0, -1), end: today.AddDate(0, 0, 15),
		},
		{
			uid: "cover", email: "bea@example.test",
			start: today, end: today.AddDate(0, 0, 14),
		},
		{
			uid: "future", email: "cara@example.test",
			start: today.AddDate(0, 0, 7), end: today.AddDate(0, 0, 9),
		},
	}
	for _, event := range events {
		if err := addEvent(calendarPath, event); err != nil {
			return err
		}
	}
	return nil
}

func addAddressBook(homePath, name string) error {
	const collection = `<?xml version="1.0"?>
<d:mkcol xmlns:d="DAV:" xmlns:a="urn:ietf:params:xml:ns:carddav">
  <d:set>
    <d:prop>
      <d:resourcetype><d:collection/><a:addressbook/></d:resourcetype>
    </d:prop>
  </d:set>
</d:mkcol>`
	return sendDAVRequest("MKCOL", homePath+name+"/", collection, "application/xml; charset=utf-8", http.StatusCreated)
}

func addCalendar(homePath, name string) error {
	return sendDAVRequest("MKCALENDAR", homePath+name+"/", "", "application/xml; charset=utf-8", http.StatusCreated)
}

func addCard(addressBookPath string, contact testContact) error {
	return putCard(addressBookPath, contact, http.StatusCreated, http.StatusNoContent)
}

func updateCard(addressBookPath string, contact testContact) error {
	return putCard(addressBookPath, contact, http.StatusOK, http.StatusNoContent)
}

func putCard(addressBookPath string, contact testContact, expected ...int) error {
	return sendDAVRequest("PUT", addressBookPath+contact.uid+".vcf",
		card(contact.uid, contact.name, contact.phone, contact.email),
		"text/vcard", expected...)
}

func addEvent(calendarPath string, event testEvent) error {
	return sendDAVRequest("PUT", calendarPath+event.uid+".ics",
		calendarEvent(event.uid, event.start, event.end, event.email),
		"text/calendar", http.StatusCreated, http.StatusNoContent)
}

func card(uid, name, phone, email string) string {
	return strings.Join([]string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"UID:" + uid,
		"N:Test;" + name + ";;;",
		"FN:" + name + " Test",
		"TEL:" + phone,
		"EMAIL:" + email,
		"END:VCARD",
		"",
	}, "\r\n")
}

func calendarEvent(uid string, start, end time.Time, email string) string {
	return strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Hilfmal integration test//EN",
		"BEGIN:VEVENT",
		"UID:" + uid,
		"DTSTAMP:20260101T000000Z",
		"CREATED:20260101T000000Z",
		"DTSTART;VALUE=DATE:" + start.Format("20060102"),
		"DTEND;VALUE=DATE:" + end.Format("20060102"),
		"SUMMARY:Testdienst",
		"ATTENDEE;PARTSTAT=ACCEPTED:mailto:" + email,
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n")
}
