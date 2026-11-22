package contacts

import (
	"fmt"
	"net/mail"
	"sort"
	"strings"

	"oberfeld.be/hilfmal/pkg/models"
)

// Contacts checks contact details independently of the duty schedule.
type Contacts struct{}

// Issues reports missing fields for all contacts.
func (c Contacts) Issues(contacts []models.Contact) []string {
	var issues []string
	for _, contact := range contacts {
		if issue := c.Issue(contact); issue != "" {
			issues = append(issues, issue+".")
		}
	}
	sort.Strings(issues)
	return issues
}

// Issue describes missing fields for a contact, or returns an empty string.
func (Contacts) Issue(contact models.Contact) string {
	var missing []string
	if strings.TrimSpace(contact.Name) == "" {
		missing = append(missing, "Vorname fehlt")
	}
	if strings.TrimSpace(contact.LastName) == "" {
		missing = append(missing, "Nachname fehlt")
	}
	if strings.TrimSpace(contact.Phone) == "" {
		missing = append(missing, "Telefonnummer fehlt")
	}
	hasEmail := false
	for _, address := range append([]string{contact.Email}, contact.AdditionalEmails...) {
		hasEmail = hasEmail || strings.TrimSpace(address) != ""
	}
	if !hasEmail {
		missing = append(missing, "E-Mail-Adresse fehlt")
	}
	if len(missing) == 0 {
		return ""
	}

	name := strings.TrimSpace(contact.DisplayName)
	if name == "" {
		name = strings.TrimSpace(strings.TrimSpace(contact.Name) + " " + strings.TrimSpace(contact.LastName))
	}
	if name == "" {
		name = "(ohne Namen)"
	}
	identifier := fmt.Sprintf("%q", name)
	for _, address := range append([]string{contact.Email}, contact.AdditionalEmails...) {
		if ValidEmail(address) {
			identifier += fmt.Sprintf(" (%q)", address)
			break
		}
	}
	if strings.TrimSpace(contact.UID) != "" {
		identifier += fmt.Sprintf(" (UID: %q)", contact.UID)
	}
	return fmt.Sprintf("Kontakt %s: %s", identifier, strings.Join(missing, ", "))
}

// ValidEmail reports whether an address is a valid standalone email address.
func ValidEmail(email string) bool {
	if strings.TrimSpace(email) != email || strings.ContainsAny(email, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}
