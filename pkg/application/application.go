package application

import (
	"errors"
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

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
	gapInterval      string
	reminderTemplate string
	adminTemplate    string
	reminderSubject  string
	adminSubject     string
}

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
	if !validEmail(sender) {
		return nil, fmt.Errorf("invalid sender email address")
	}
	if !validEmail(emergency) {
		return nil, fmt.Errorf("invalid emergency email address")
	}

	if reminderDays < 0 {
		return nil, fmt.Errorf("reminder days must be non-negative")
	}

	// Validate gap interval format (e.g., P1M, P1W, P7D)
	if !isValidDateInterval(gapInterval) {
		return nil, fmt.Errorf("invalid gap interval format")
	}

	return &Application{
		source:           source,
		mailer:           mail,
		publisher:        pub,
		sender:           sender,
		emergency:        emergency,
		reminderDays:     reminderDays,
		gapInterval:      gapInterval,
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
		if !validEmail(email) {
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

	gapEnd, err := a.addDateInterval(today, a.gapInterval)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}

	if gapEnd.Before(today) || gapEnd.Equal(today) {
		return time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("gap interval must be greater than zero")
	}

	return today, target, gapEnd, nil
}

func (a *Application) addDateInterval(t time.Time, interval string) (time.Time, error) {
	// Parse interval like P1M, P1W, P7D
	re := regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?$`)
	matches := re.FindStringSubmatch(interval)
	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid interval format: %s", interval)
	}

	var units [4]int
	for i := 1; i < len(matches); i++ {
		if matches[i] == "" {
			continue
		}
		value, err := strconv.Atoi(matches[i])
		if err != nil {
			return time.Time{}, fmt.Errorf("ungültiges Datumsintervall %q: %w", interval, err)
		}
		units[i-1] = value
	}

	result := t.AddDate(units[0], units[1], units[2]*7+units[3])
	return result, nil
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

func validEmail(email string) bool {
	if strings.TrimSpace(email) != email || strings.ContainsAny(email, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func isValidDateInterval(interval string) bool {
	re := regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?$`)
	return interval != "P" && re.MatchString(interval)
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
