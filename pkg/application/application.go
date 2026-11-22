package application

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	contactcheck "oberfeld.be/hilfmal/pkg/contacts"
	"oberfeld.be/hilfmal/pkg/datasource"
	"oberfeld.be/hilfmal/pkg/mailer"
	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/planner"
	"oberfeld.be/hilfmal/pkg/publisher"
)

// Application coordinates the main logic
type Application struct {
	source           datasource.DataSource
	mailer           mailer.Mailer
	publisher        publisher.Publisher
	sender           string
	emergency        string
	reminderDays     int
	gapInterval      dateInterval
	reminderTemplate string
	adminTemplate    string
	reminderSubject  string
	adminSubject     string
}

type dateInterval struct {
	years  int
	months int
	weeks  int
	days   int
}

var dateIntervalPattern = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?$`)

// NewApplication creates a new application instance
func NewApplication(
	source datasource.DataSource,
	mail mailer.Mailer,
	pub publisher.Publisher,
	sender,
	emergency,
	reminderTemplate,
	adminTemplate,
	reminderSubject,
	adminSubject string,
	reminderDays int,
	gapInterval string,
) (*Application, error) {
	// Validate email addresses
	if !contactcheck.ValidEmail(sender) {
		return nil, fmt.Errorf("invalid sender email address")
	}
	if !contactcheck.ValidEmail(emergency) {
		return nil, fmt.Errorf("invalid emergency email address")
	}

	if reminderDays < 0 {
		return nil, fmt.Errorf("reminder days must be non-negative")
	}

	parsedGapInterval, err := parseDateInterval(gapInterval)
	if err != nil {
		return nil, err
	}

	return &Application{
		source:           source,
		mailer:           mail,
		publisher:        pub,
		sender:           sender,
		emergency:        emergency,
		reminderDays:     reminderDays,
		gapInterval:      parsedGapInterval,
		reminderTemplate: reminderTemplate,
		adminTemplate:    adminTemplate,
		reminderSubject:  reminderSubject,
		adminSubject:     adminSubject,
	}, nil
}

// Run executes a daily run and returns 0 on success, 1 on error
// Admin lookup and email errors do not prevent publication.
func (a *Application) Run(now time.Time) int {
	var admins []string
	stage := "Administrationsadressen lesen"

	if err := a.run(now, &admins, &stage); err != nil {
		a.reportFailure(err, stage, admins)
		return 1
	}

	return 0
}

// PublishOnly updates the public page without sending email.
func (a *Application) PublishOnly(now time.Time) error {
	p := planner.NewPlanner()
	date, err := p.ServiceDate(now)
	if err != nil {
		return fmt.Errorf("Diensttag berechnen: %w", err)
	}
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return fmt.Errorf("Schweizer Zeitzone laden: %w", err)
	}
	start, err := time.ParseInLocation("2006-01-02", date, location)
	if err != nil {
		return fmt.Errorf("Diensttag parsen: %w", err)
	}
	events, err := a.source.Events(start, start.AddDate(0, 0, 1))
	if err != nil {
		return fmt.Errorf("Kalender lesen: %w", err)
	}
	contacts, err := a.source.Contacts()
	if err != nil {
		return fmt.Errorf("Mitwirkende lesen: %w", err)
	}
	issues := append(append([]string(nil), a.source.Issues()...), p.Issues(events, contacts)...)
	issues = append(issues, (contactcheck.Contacts{}).Issues(contacts)...)
	for _, issue := range issues {
		log.Printf("Hilfmal: Hinweis bei Veröffentlichung: %s", issue)
	}
	if err := a.publisher.Publish(p.Select(date, events, contacts)); err != nil {
		return fmt.Errorf("Webseite veröffentlichen: %w", err)
	}
	return nil
}

func (a *Application) run(now time.Time, admins *[]string, stage *string) error {
	var err error
	var runErr error
	errorStage := ""
	*admins, err = a.loadAdmins()
	if err != nil {
		*stage = "Administrationsadressen lesen"
		runErr = err
		errorStage = *stage
	}

	planner := planner.NewPlanner()
	today, target, gapEnd, err := a.calculateDateRange(now, planner)
	if err != nil {
		*stage = "Datumsbereich berechnen"
		return errors.Join(runErr, err)
	}

	*stage = "Kalender lesen"
	events, err := a.loadEvents(today, target, gapEnd)
	if err != nil {
		return errors.Join(runErr, err)
	}

	*stage = "Mitwirkende lesen"
	contacts, err := a.source.Contacts()
	if err != nil {
		return errors.Join(runErr, err)
	}

	current := planner.Select(today.Format("2006-01-02"), events, contacts)

	reminder := planner.Reminder(target.Format("2006-01-02"), events, contacts)

	if reminder != nil {
		*stage = "Erinnerung versenden"
		if err := a.sendReminder(reminder, target); err != nil {
			if errorStage == "" {
				errorStage = *stage
			}
			runErr = errors.Join(runErr, err)
		}
	}

	issues := a.collectIssues(today, gapEnd, planner, events, contacts)
	if len(issues) > 0 {
		*stage = "Administrationshinweis versenden"
		if err := a.sendAdminNotices(*admins, issues); err != nil {
			if errorStage == "" {
				errorStage = *stage
			}
			runErr = errors.Join(runErr, err)
		}
	}

	*stage = "Webseite veröffentlichen"
	publishErr := a.publisher.Publish(current)
	if publishErr != nil {
		return errors.Join(runErr, publishErr)
	}
	if runErr != nil {
		*stage = errorStage
		return runErr
	}
	return nil
}

func (a *Application) loadAdmins() ([]string, error) {
	admins, err := a.source.Admins()
	if err != nil {
		return nil, err
	}

	if len(admins) == 0 {
		return nil, fmt.Errorf("admin addressbook contains no valid recipients")
	}

	for _, email := range admins {
		if !contactcheck.ValidEmail(email) {
			return nil, fmt.Errorf("Administrationsadressbuch enthält ungültige Empfänger.")
		}
	}

	return admins, nil
}

func (a *Application) calculateDateRange(now time.Time, p *planner.Planner) (time.Time, time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("Schweizer Zeitzone laden: %w", err)
	}
	now = now.In(loc)
	serviceDay, err := p.ServiceDate(now)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}

	parsedDay, err := time.Parse("2006-01-02", serviceDay)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("Diensttag parsen: %w", err)
	}
	year, month, day := parsedDay.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, loc)

	target := today.AddDate(0, 0, a.reminderDays)

	gapEnd := a.gapInterval.addTo(today)

	return today, target, gapEnd, nil
}

func parseDateInterval(value string) (dateInterval, error) {
	matches := dateIntervalPattern.FindStringSubmatch(value)
	if matches == nil {
		return dateInterval{}, fmt.Errorf("invalid interval format: %s", value)
	}

	var interval dateInterval
	units := []*int{&interval.years, &interval.months, &interval.weeks, &interval.days}
	for i := 1; i < len(matches); i++ {
		if matches[i] == "" {
			continue
		}
		value, err := strconv.Atoi(matches[i])
		if err != nil {
			return dateInterval{}, fmt.Errorf("ungültiges Datumsintervall: %w", err)
		}
		*units[i-1] = value
	}

	if interval == (dateInterval{}) {
		return dateInterval{}, fmt.Errorf("gap interval must be greater than zero")
	}

	return interval, nil
}

func (interval dateInterval) addTo(t time.Time) time.Time {
	return t.AddDate(interval.years, interval.months, interval.weeks*7+interval.days)
}

func (a *Application) loadEvents(today, target, gapEnd time.Time) ([]models.Event, error) {
	end := target.AddDate(0, 0, 1)
	if gapEnd.After(end) {
		end = gapEnd
	}

	return a.source.Events(today.AddDate(0, 0, -1), end)
}

func (a *Application) sendReminder(contact *models.Contact, target time.Time) error {
	body := a.reminderTemplate
	body = strings.ReplaceAll(body, "{{name}}", contact.Name)
	body = strings.ReplaceAll(body, "{{date}}", target.Format("2006-01-02"))

	return a.mailer.Send(contact.Email, a.sender, a.reminderSubject, body)
}

func (a *Application) collectIssues(today, gapEnd time.Time, p *planner.Planner, events []models.Event, contacts []models.Contact) []string {
	sourceIssues := a.source.Issues()
	plannerIssues := p.Issues(events, contacts)
	gaps := p.Gaps(today.Format("2006-01-02"), gapEnd.Format("2006-01-02"), events, contacts)

	var issues []string
	issues = append(issues, sourceIssues...)
	issues = append(issues, plannerIssues...)
	issues = append(issues, (contactcheck.Contacts{}).Issues(contacts)...)

	for _, gap := range gaps {
		issues = append(issues, fmt.Sprintf("Kein gültiger Dienst: %s", gap))
	}

	return issues
}

func (a *Application) sendAdminNotices(admins []string, issues []string) error {
	body := a.adminTemplate
	body = strings.ReplaceAll(body, "{{issues}}", strings.Join(issues, "\n"))

	for _, admin := range uniqueStrings(admins) {
		if err := a.mailer.Send(admin, a.sender, a.adminSubject, body); err != nil {
			return err
		}
	}

	return nil
}

func (a *Application) reportFailure(err error, stage string, admins []string) {
	log.Printf("Hilfmal: Run failed at stage '%s': %v", stage, err)

	recipients := append(append([]string(nil), admins...), a.emergency)
	for _, recipient := range uniqueStrings(recipients) {
		if err := a.mailer.Send(recipient, a.sender, "Fehler beim Hilfmal-Programmlauf",
			"Der Hilfmal-Programmlauf ist fehlgeschlagen. Bitte das Serverprotokoll und die Konfiguration prüfen."); err != nil {
			log.Printf("Hilfmal: Failed to send error notification: %v", err)
		}
	}
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
