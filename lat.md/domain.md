# Domain

The domain model resolves an overlapping, recurring Nextcloud calendar against a unique contact key, then presents at most one valid person for one Swiss-local duty day.

## Duty day

A duty begins at 06:00 in `Europe/Zurich` and ends at 06:00 the following day. A run between midnight and 06:00 belongs to the previous service day, as defined by [[pkg/planner/planner.go#Planner#ServiceDate]].

Calendar entries are all-day iCalendar events with half-open date ranges. A multi-day entry covers each date it spans, and recurring occurrences and recurrence exceptions are expanded by [[pkg/parser/parser.go#CalendarParser#Parse]]. DTSTART, DTEND or DURATION must represent valid all-day dates.

## Attendee and contact identity
Calendar attendee and organizer emails and event titles identify Mitwirkende. Resolution ignores RSVP status and requires a complete contact.

Within an entry, [[pkg/planner/planner.go#Planner#resolve]] prefers a valid attendee other than the organizer, then a name in the title, then the organizer. An unresolved ambiguous title blocks organizer fallback and produces an administrative issue.

Title matching is case-insensitive and uses complete names at word boundaries. When several complete contacts have the same matching first name and no family name is stated, or share the same matching full name, the first valid email in alphabetical order determines the selection and an administrative warning identifies it. Identical email addresses are ordered by phone number. Other ambiguous names remain unresolved. A contact with several email addresses uses its first valid address for title matching; attendee matching retains the address supplied by the calendar.

The public page is therefore a view of identity and reachability, not a general export of calendar or address-book data. Only name and phone are rendered, with a telephone link; [[pkg/publisher/publisher.go#FilePublisher#render]] escapes the values.

## Overlap priority

For a duty date, [[pkg/planner/planner.go#Planner#Select]] considers only entries that cover that date. It chooses:
1. the shortest covering entry;
2. for equal duration, the latest CREATED timestamp;
3. when creation-time information does not decide, the alphanumerically smallest UID.

If a duration group has a missing CREATED value, creation time is skipped for the whole equal-duration group and UID decides. This makes tie-breaking independent of input order. The organizer fallback applies within each entry, so it does not override the ordering between entries. If the highest-priority entry cannot resolve to a complete contact, selection continues to the next valid entry.

## Coverage and notices

[[pkg/planner/planner.go#Planner#Gaps]] reports a day as uncovered when no valid contact can be selected. The default future check is one month and is configurable through an ISO date interval such as P1M.

[[pkg/planner/planner.go#Planner#Reminder]] sends a reminder once for the first day of a continuous duty held by the same person. Invalid calendar entries and unresolved contacts become administrative issues rather than silently becoming an assignment.

The application combines source issues, planner issues and uncovered days before emailing the administration. This behavior is coordinated by [[pkg/application/application.go#Application#collectIssues]].

For an unresolved contact, [[pkg/planner/planner.go#Planner#Issues]] identifies the duty and reports matching incomplete contacts and their missing or invalid required fields. Contacts without email are retained for diagnosis but remain ineligible for assignment.

An invalid or missing start date is reported with the event title, UID and available start value so the entry can be found; a timestamp retains its stated timezone.
