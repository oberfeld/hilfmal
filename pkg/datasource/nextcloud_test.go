package datasource

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNextcloudSourceReadsCalendarContactsAndAdmins(t *testing.T) {
	source := NewNextcloudSource(fakeDAV{}, "calendar", "contacts", "admins")

	admins, err := source.Admins()
	if err != nil || len(admins) != 1 || admins[0] != "admin@example.test" {
		t.Fatalf("Admins() = %v, %v", admins, err)
	}
	contacts, err := source.Contacts()
	if err != nil || len(contacts) != 1 {
		t.Fatalf("Contacts() = %v, %v", contacts, err)
	}
	if contacts[0].Name != "Ada" || contacts[0].Phone != "+41 12" || contacts[0].Email != "ada@example.test" {
		t.Fatalf("contact = %#v", contacts[0])
	}
	events, err := source.Events(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || len(events) != 1 || events[0].Attendees[0] != "ada@example.test" {
		t.Fatalf("Events() = %v, %v", events, err)
	}
}

func TestNextcloudSourceRejectsDAVPropertyErrors(t *testing.T) {
	source := NewNextcloudSource(fakeDAV{broken: true}, "calendar", "contacts", "admins")
	if _, err := source.Contacts(); err == nil {
		t.Fatal("expected property status error")
	}
}

func TestNextcloudSourceRejectsMissingDAVStatus(t *testing.T) {
	source := NewNextcloudSource(fakeDAV{
		response: `<d:multistatus xmlns:d="DAV:" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:response><d:propstat><d:prop><a:address-data>value</a:address-data></d:prop></d:propstat></d:response></d:multistatus>`,
	}, "calendar", "contacts", "admins")
	if _, err := source.Contacts(); err == nil {
		t.Fatal("expected empty addressbook data error")
	}
}

func TestHTTPDAVTransportRequiresHTTPS(t *testing.T) {
	transport := NewHTTPDAVTransport("user", "password")
	if _, err := transport.Report("http://cloud.example.test/calendar", "<report/>"); err == nil {
		t.Fatal("expected insecure DAV URL to fail")
	}
}

func TestHTTPDAVTransportRejectsMalformedAndHostlessURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "malformed", url: "https://%", want: "invalid DAV URL"},
		{name: "missing host", url: "https:///calendar", want: "DAV access requires HTTPS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewHTTPDAVTransport("user", "password")
			if _, err := transport.Report(tt.url, "<report/>"); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Report() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestHTTPDAVTransportRejectsUnexpectedStatus(t *testing.T) {
	transport := NewHTTPDAVTransport("user", "password")
	transport.client = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	if _, err := transport.Report("https://cloud.example.test/calendar", "<report/>"); err == nil ||
		!strings.Contains(err.Error(), "HTTP 200") {
		t.Fatalf("Report() error = %v, want unexpected HTTP status", err)
	}
}

func TestIsSuccessfulDAVStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "HTTP/1.1 200 OK", want: true},
		{status: "HTTP/1.1 299 Multi-Status", want: true},
		{status: "HTTP/1.1 200", want: true},
		{status: "HTTP/1.1 199 Early", want: false},
		{status: "HTTP/1.1 300 Multiple Choices", want: false},
		{status: "HTTP/1.1", want: false},
		{status: "NOTHTTP/1.1 200 OK", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isSuccessfulDAVStatus(tt.status); got != tt.want {
				t.Fatalf("isSuccessfulDAVStatus(%q) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type fakeDAV struct {
	broken   bool
	response string
}

func (d fakeDAV) Report(resource, _ string) (string, error) {
	if d.response != "" {
		return d.response, nil
	}
	if d.broken {
		return `<d:multistatus xmlns:d="DAV:"><d:response><d:propstat><d:prop/><d:status>HTTP/1.1 403 Forbidden</d:status></d:propstat></d:response></d:multistatus>`, nil
	}
	var data string
	switch resource {
	case "admins":
		data = "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Admin;Admin;;;\r\nEMAIL:admin@example.test\r\nEND:VCARD\r\n"
	case "contacts":
		data = "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Lovelace;Ada;;;\r\nFN:Ada Lovelace\r\nTEL:+41 12\r\nEMAIL:ada@example.test\r\nEND:VCARD\r\n"
	case "calendar":
		data = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:test\r\nDTSTART;VALUE=DATE:20261101\r\n" +
			"DTEND;VALUE=DATE:20261102\r\nATTENDEE:mailto:ada@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	default:
		return "", fmt.Errorf("unknown resource")
	}
	tag := "a:address-data"
	if resource == "calendar" {
		tag = "c:calendar-data"
	}
	return fmt.Sprintf(
		`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:response><d:propstat><d:prop><%s>%s</%s></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`,
		tag, html.EscapeString(data), tag,
	), nil
}
