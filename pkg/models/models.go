package models

// Event represents a calendar event
type Event struct {
	UID       string   // Unique identifier
	Start     string   // Date in YYYY-MM-DD format (inclusive)
	End       string   // Date in YYYY-MM-DD format (exclusive)
	Summary   string   // Calendar event title
	Attendees []string // Email addresses of attendees
	Organizer string   // Email address of the organizer
	Created   *string  // ISO8601 timestamp, nil if not set
}

// Contact represents a person in the contacts list
type Contact struct {
	Name             string   // First name
	LastName         string   // Family name
	Phone            string   // Phone number
	Email            string   // First email address
	AdditionalEmails []string // Other email addresses for the same person
}

// Config represents the application configuration
type Config struct {
	DAVUser          string `yaml:"dav_user"`
	DAVPassword      string `yaml:"dav_password"`
	CalendarURL      string `yaml:"calendar_url"`
	ContactsURL      string `yaml:"contacts_url"`
	AdminsURL        string `yaml:"admins_url"`
	Sender           string `yaml:"sender"`
	EmergencyEmail   string `yaml:"emergency_email"`
	ReminderDays     int    `yaml:"reminder_days"`
	GapInterval      string `yaml:"gap_interval"`
	PublicPage       string `yaml:"public_page"`
	WebTemplate      string `yaml:"web_template"`
	ReminderTemplate string `yaml:"reminder_template"`
	AdminTemplate    string `yaml:"admin_template"`
	ReminderSubject  string `yaml:"reminder_subject"`
	AdminSubject     string `yaml:"admin_subject"`
}
