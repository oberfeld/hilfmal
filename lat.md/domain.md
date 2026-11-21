# Domain

The domain model resolves an overlapping, recurring Nextcloud calendar against a unique contact key, then presents at most one valid person for one Swiss-local duty day.

## Duty day

A duty begins at 06:00 in `Europe/Zurich` and ends at 06:00 the following day. A run between midnight and 06:00 belongs to the previous service day, as defined by [[pkg/planner/planner.go#Planner#ServiceDate]].

Calendar entries are all-day iCalendar events with half-open date ranges. A multi-day entry covers each date it spans, and recurring occurrences and recurrence exceptions are expanded by [[pkg/parser/parser.go#CalendarParser#Parse]]. DTSTART, DTEND or DURATION must represent valid all-day dates.

## Attendee and contact identity
The calendar attendee email is the stable join key to Mitwirkende. It ignores RSVP status and requires a unique, complete contact.

The public page is therefore a view of identity and reachability, not a general export of calendar or address-book data. Only name and phone are rendered; the implementation escapes both values in [[pkg/publisher/publisher.go#FilePublisher#render]].

## Overlap priority

For a duty date, [[pkg/planner/planner.go#Planner#Select]] considers only entries that cover that date. It chooses:
1. the shortest covering entry;
2. for equal duration, the latest CREATED timestamp;
3. when creation-time information does not decide, the alphanumerically smallest UID.

If a duration group has a missing CREATED value, creation time is skipped for the whole equal-duration group and UID decides. This makes tie-breaking independent of input order. If the highest-priority entry cannot resolve to a complete contact, selection continues to the next valid entry.

## Coverage and notices

[[pkg/planner/planner.go#Planner#Gaps]] reports a day as uncovered when no valid contact can be selected. The default future check is one month and is configurable through an ISO date interval such as P1M.

[[pkg/planner/planner.go#Planner#Reminder]] sends a reminder once for the first day of a continuous duty held by the same person. Invalid calendar entries and unresolved contacts become administrative issues rather than silently becoming an assignment.

The application combines source issues, planner issues and uncovered days before emailing the administration. This behavior is coordinated by [[pkg/application/application.go#Application#collectIssues]].
