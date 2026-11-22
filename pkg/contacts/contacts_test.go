package contacts

import (
	"strings"
	"testing"

	"oberfeld.be/hilfmal/pkg/models"
	"oberfeld.be/hilfmal/pkg/parser"
)

func TestContactIssuesCheckEveryContactWithoutReportingInvalidEmails(t *testing.T) {
	cards := []string{
		"BEGIN:VCARD\nUID:phone-missing\nN:Beispiel;Karin;;;\nFN:Karin Beispiel\nEMAIL:karin@example.test\nEND:VCARD\n",
		"BEGIN:VCARD\nUID:name-missing\nFN:Unvollständiger Kontakt\nTEL:123\nEND:VCARD\n",
		"BEGIN:VCARD\nUID:complete\nN:Beispiel;Ada;;;\nTEL:123\nEMAIL:invalid\nEMAIL:ada@example.test\nEND:VCARD\n",
		"BEGIN:VCARD\nUID:invalid-only\nN:Beispiel;Berta;;;\nTEL:456\nEMAIL:invalid\nEND:VCARD\n",
	}
	var contacts []models.Contact
	for _, card := range cards {
		parsed, err := parser.ParseVCard(card)
		if err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, parsed...)
	}
	contacts = append(contacts, models.Contact{Name: "Clara", Phone: "789", Email: "clara@example.test"})
	issues := (Contacts{}).Issues(contacts)
	if len(issues) != 3 {
		t.Fatalf("Issues() = %v, want three incomplete contacts", issues)
	}
	text := strings.Join(issues, "\n")
	for _, want := range []string{"Karin Beispiel", "phone-missing", "Telefonnummer fehlt", "Unvollständiger Kontakt", "name-missing", "Vorname fehlt", "Nachname fehlt", "E-Mail-Adresse fehlt", "Clara"} {
		if !strings.Contains(text, want) {
			t.Errorf("issues %q do not contain %q", text, want)
		}
	}
	for _, unwanted := range []string{"invalid", "Ada", "Berta"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("issues %q unexpectedly contain %q", text, unwanted)
		}
	}
}
