package planner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"oberfeld.be/hilfmal/pkg/models"
)

type contactScenarioKey struct{}

type contactScenario struct {
	contacts []models.Contact
	events   []models.Event
	selected *models.Contact
	issues   []string
}

func TestContactFeatures(t *testing.T) {
	suite := godog.TestSuite{
		Name:                "kontaktzuordnung",
		ScenarioInitializer: initializeContactScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/kontaktzuordnung.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("Godog scenarios failed")
	}
}

func initializeContactScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return context.WithValue(ctx, contactScenarioKey{}, &contactScenario{}), nil
	})
	ctx.Step(`^das Adressbuch enthält$`, contactBookContains)
	ctx.Step(`^ein Dienst mit Titel "([^"]*)", Organisator "([^"]*)" und Teilnehmenden "([^"]*)"$`, contactDuty)
	ctx.Step(`^ein kurzer Dienst mit Organisator "([^"]*)" und ein längerer Dienst mit teilnehmender Person "([^"]*)"$`, overlappingContactDuties)
	ctx.Step(`^die zuständige Person ermittelt wird$`, resolveContactScenario)
	ctx.Step(`^ist die E-Mail der zuständigen Person "([^"]*)"$`, selectedContactEmail)
	ctx.Step(`^wird keine Person zugeordnet$`, noContactSelected)
	ctx.Step(`^es gibt keinen Kontakt-Hinweis$`, noContactIssue)
	ctx.Step(`^es gibt einen Kontakt-Hinweis$`, hasContactIssue)
	ctx.Step(`^der Kontakt-Hinweis enthält "([^"]+)"$`, contactIssueContains)
}

func contactScenarioFrom(ctx context.Context) (*contactScenario, error) {
	scenario, ok := ctx.Value(contactScenarioKey{}).(*contactScenario)
	if !ok {
		return nil, errors.New("contact scenario is not initialized")
	}
	return scenario, nil
}

func contactBookContains(ctx context.Context, table *godog.Table) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.contacts = nil
	for _, row := range table.Rows[1:] {
		if len(row.Cells) != 4 {
			return fmt.Errorf("contact row must have four columns, got %d", len(row.Cells))
		}
		emails := splitAddresses(row.Cells[3].Value)
		contact := models.Contact{Name: row.Cells[0].Value, LastName: row.Cells[1].Value, Phone: row.Cells[2].Value}
		if len(emails) > 0 {
			contact.Email = emails[0]
			contact.AdditionalEmails = emails[1:]
		}
		scenario.contacts = append(scenario.contacts, contact)
	}
	return nil
}

func splitAddresses(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	addresses := strings.Split(value, ",")
	for i := range addresses {
		addresses[i] = strings.TrimSpace(addresses[i])
	}
	return addresses
}

func contactDuty(ctx context.Context, title, organizer, attendees string) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.events = []models.Event{{
		UID: "dienst", Start: "2026-11-01", End: "2026-11-02", Summary: title,
		Organizer: organizer, Attendees: splitAddresses(attendees),
	}}
	return nil
}

func overlappingContactDuties(ctx context.Context, organizer, attendee string) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	scenario.events = []models.Event{
		{UID: "lang", Start: "2026-11-01", End: "2026-11-05", Attendees: []string{attendee}},
		{UID: "kurz", Start: "2026-11-01", End: "2026-11-02", Organizer: organizer},
	}
	return nil
}

func resolveContactScenario(ctx context.Context) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	p := NewPlanner()
	scenario.selected = p.Select("2026-11-01", scenario.events, scenario.contacts)
	scenario.issues = p.Issues(scenario.events, scenario.contacts)
	return nil
}

func selectedContactEmail(ctx context.Context, email string) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	if scenario.selected == nil || scenario.selected.Email != email {
		return fmt.Errorf("selected contact = %#v, want email %q", scenario.selected, email)
	}
	return nil
}

func noContactSelected(ctx context.Context) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	if scenario.selected != nil {
		return fmt.Errorf("unexpected selected contact: %#v", scenario.selected)
	}
	return nil
}

func noContactIssue(ctx context.Context) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	if len(scenario.issues) != 0 {
		return fmt.Errorf("unexpected contact issues: %v", scenario.issues)
	}
	return nil
}

func hasContactIssue(ctx context.Context) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	if len(scenario.issues) != 1 || !strings.Contains(scenario.issues[0], "dienst") {
		return fmt.Errorf("expected a contact issue for dienst, got %v", scenario.issues)
	}
	return nil
}

func contactIssueContains(ctx context.Context, text string) error {
	scenario, err := contactScenarioFrom(ctx)
	if err != nil {
		return err
	}
	if len(scenario.issues) != 1 || !strings.Contains(scenario.issues[0], text) {
		return fmt.Errorf("contact issue = %v, want text %q", scenario.issues, text)
	}
	return nil
}
