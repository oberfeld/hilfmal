package datasource

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/parser"
)

// DAVTransport handles DAV protocol communication
type DAVTransport interface {
	Report(url, body string) (string, error)
}

// HTTPDAVTransport implements DAVTransport using HTTPS.
type HTTPDAVTransport struct {
	user     string
	password string
	client   *http.Client
}

func NewHTTPDAVTransport(user, password string) *HTTPDAVTransport {
	return &HTTPDAVTransport{
		user:     user,
		password: password,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (t *HTTPDAVTransport) Report(davURL, body string) (string, error) {
	parsed, err := url.Parse(davURL)
	if err != nil {
		return "", fmt.Errorf("invalid DAV URL: %w", err)
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("DAV access requires HTTPS")
	}

	req, err := http.NewRequest("REPORT", davURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create DAV request: %w", err)
	}

	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.SetBasicAuth(t.user, t.password)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("DAV request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		return "", fmt.Errorf("DAV request failed (HTTP %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read DAV response: %w", err)
	}

	return string(content), nil
}

// NextcloudSource implements DataSource for Nextcloud
type NextcloudSource struct {
	transport   DAVTransport
	calendarURL string
	contactsURL string
	adminsURL   string
	issues      []string
}

// NewNextcloudSource creates a new Nextcloud data source
func NewNextcloudSource(transport DAVTransport, calendarURL, contactsURL, adminsURL string) *NextcloudSource {
	return &NextcloudSource{
		transport:   transport,
		calendarURL: calendarURL,
		contactsURL: contactsURL,
		adminsURL:   adminsURL,
		issues:      []string{},
	}
}

// query retrieves DAV data and parses the XML response
func (s *NextcloudSource) query(davURL string, isCalendar bool) ([]string, error) {
	var body string
	if isCalendar {
		body = `<?xml version="1.0"?><c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav"><d:prop><c:calendar-data/></d:prop><c:filter><c:comp-filter name="VCALENDAR"><c:comp-filter name="VEVENT"/></c:comp-filter></c:filter></c:calendar-query>`
	} else {
		body = `<?xml version="1.0"?><a:addressbook-query xmlns:d="DAV:" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:prop><a:address-data/></d:prop></a:addressbook-query>`
	}

	response, err := s.transport.Report(davURL, body)
	if err != nil {
		return nil, err
	}

	type Multistatus struct {
		XMLName   xml.Name `xml:"multistatus"`
		Responses []struct {
			Status   string `xml:"status"`
			Propstat []struct {
				Prop struct {
					CalendarData *string `xml:"calendar-data"`
					AddressData  *string `xml:"address-data"`
				} `xml:"prop"`
				Status string `xml:"status"`
			} `xml:"propstat"`
		} `xml:"response"`
	}

	var ms Multistatus
	if err := xml.Unmarshal([]byte(response), &ms); err != nil {
		return nil, fmt.Errorf("invalid DAV response: %w", err)
	}
	if ms.XMLName.Local != "multistatus" || ms.XMLName.Space != "DAV:" {
		return nil, fmt.Errorf("invalid DAV response root")
	}

	var result []string
	for _, resp := range ms.Responses {
		statuses := make([]string, 0, len(resp.Propstat)+1)
		if resp.Status != "" {
			statuses = append(statuses, resp.Status)
		}
		for _, propstat := range resp.Propstat {
			statuses = append(statuses, propstat.Status)
		}
		if len(statuses) == 0 {
			return nil, fmt.Errorf("DAV response is missing a status")
		}
		for _, status := range statuses {
			if !isSuccessfulDAVStatus(status) {
				return nil, fmt.Errorf("DAV resource could not be read")
			}
		}

		var data []string
		for _, ps := range resp.Propstat {
			if isCalendar && ps.Prop.CalendarData != nil {
				data = append(data, *ps.Prop.CalendarData)
			} else if !isCalendar && ps.Prop.AddressData != nil {
				data = append(data, *ps.Prop.AddressData)
			}
		}
		if len(data) != 1 {
			return nil, fmt.Errorf("DAV response has no unique requested data")
		}
		result = append(result, data[0])
	}

	return result, nil
}

func isSuccessfulDAVStatus(status string) bool {
	fields := strings.Fields(status)
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
		return false
	}
	code, err := strconv.Atoi(fields[1])
	return err == nil && code >= 200 && code < 300
}

// Admins returns admin email addresses
func (s *NextcloudSource) Admins() ([]string, error) {
	data, err := s.query(s.adminsURL, false)
	if err != nil {
		return nil, err
	}

	var emails []string
	for _, vcardData := range data {
		contactEmails, err := parser.ParseVCardEmails(vcardData)
		if err != nil {
			return nil, fmt.Errorf("invalid admin contact: %w", err)
		}
		emails = append(emails, contactEmails...)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("admin addressbook contains no valid recipients")
	}

	return emails, nil
}

// Contacts returns all contacts
func (s *NextcloudSource) Contacts() ([]models.Contact, error) {
	data, err := s.query(s.contactsURL, false)
	if err != nil {
		return nil, err
	}

	var contacts []models.Contact
	for _, vcardData := range data {
		vcardContacts, err := parser.ParseVCard(vcardData)
		if err != nil {
			return nil, fmt.Errorf("invalid contact: %w", err)
		}
		contacts = append(contacts, vcardContacts...)
	}

	return contacts, nil
}

// Events returns calendar events in the given time range
func (s *NextcloudSource) Events(from, to time.Time) ([]models.Event, error) {
	data, err := s.query(s.calendarURL, true)
	if err != nil {
		return nil, err
	}

	s.issues = []string{}
	calParser := parser.NewCalendarParser()

	var events []models.Event
	for _, icalData := range data {
		parsed, parserIssues, err := calParser.Parse(icalData, from, to)
		if err != nil {
			return nil, fmt.Errorf("invalid calendar data: %w", err)
		}
		events = append(events, parsed...)
		s.issues = append(s.issues, parserIssues...)
	}

	return events, nil
}

// Issues returns any issues from the last Events() call
func (s *NextcloudSource) Issues() []string {
	return s.issues
}
