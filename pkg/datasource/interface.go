package datasource

import (
	"oberfeld.be/hilfmal/pkg/models"
	"time"
)

// DataSource defines the interface for data access
type DataSource interface {
	// Admins returns a list of admin email addresses
	Admins() ([]string, error)

	// Contacts returns a list of all contacts
	Contacts() ([]models.Contact, error)

	// Events returns events in the given time range (from inclusive, to exclusive)
	Events(from, to time.Time) ([]models.Event, error)

	// Issues returns any issues found during the last Events() call
	Issues() []string
}
