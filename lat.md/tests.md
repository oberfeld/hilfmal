---
lat:
  require-code-mention: true
---
# Tests

The test suite verifies domain rules at package boundaries and the full application workflow with replaceable in-memory, local-DAV or containerized dependencies.

## Daily application workflow

Application scenarios exercise the orchestration and expected exit behavior without relying on Nextcloud or a real mail server.

### Run publication and notice behavior

The complete run scenario must publish the selected current contact, send a one-time reminder and one gap notice when applicable, and report mail or publication failures as non-zero runs.

This is specified by [[pkg/application/features/application.feature]] and implemented by [[pkg/application/application_bdd_test.go#TestApplicationFeatures]].

### Publication-only mode

The publication-only mode selects the current Swiss service day without loading administrators or sending mail, and propagates data-source and publication failures.

This is covered by [[pkg/application/application_test.go#TestPublishOnly]], [[pkg/application/application_test.go#TestSameApplicationCanPublishOnlyAndRun]], [[pkg/application/application_test.go#TestPublishOnlyWithDAVAndFilePublisher]], and [[cmd/hilfmal/main_test.go#TestPublishOnlyStartupFailureDoesNotSendEmail]].

### Swiss service date and ranges

A run at 06:00 uses the new Swiss service date, while earlier times use the previous day. Calendar reads, month horizons and DST transitions must use the Swiss local date model consistently.

This intent is covered across [[pkg/planner/planner_test.go#TestServiceDate]], [[pkg/planner/planner_test.go#TestServiceDateAtDaylightSavingTransitions]], and [[pkg/application/application_test.go#TestMonthRangeUsesSwissLocalMidnight]].

## Calendar and contact resolution

The parser and planner tests protect the meaning of duties when entries overlap or recur.

### Priority and contact validity

Shorter events win overlapping entries, and equal-duration events use later creation time or UID as a deterministic final tie-break. Missing or ambiguous contacts are not fabricated; the next valid event may be selected and issues are reported.

This intent is covered by [[pkg/planner/planner_test.go#TestSelect]] and [[pkg/planner/planner_test.go#TestPriorityAndMissingCreationTieBreakAreInputOrderIndependent]].

### Organizer fallback

Valid attendees other than the organizer take priority within an entry; otherwise an unambiguous title or a valid organizer can be selected without changing priority between overlapping entries.

This is specified by [[pkg/planner/features/kontaktzuordnung.feature]] and covered by [[pkg/planner/contact_bdd_test.go#TestContactFeatures]], [[pkg/planner/planner_test.go#TestOrganizerFallback]], and [[pkg/parser/parser_test.go#TestOrganizerOnRecurringEvents]].

### Title-based contact resolution

Calendar titles select complete contacts by first name, using family names to resolve ambiguity. Equal first names without a surname and identical full names select by email and produce a warning; other ambiguity remains unresolved.

This is specified by [[pkg/planner/features/kontaktzuordnung.feature]] and covered by [[pkg/planner/contact_bdd_test.go#TestContactFeatures]], [[pkg/planner/planner_test.go#TestSummaryFallback]], [[pkg/parser/parser_test.go#TestOrganizerOnRecurringEvents]], [[pkg/parser/parser_test.go#TestParseVCardUsesGivenNameAndEmails]], [[pkg/application/application_test.go#TestPublishOnlyWithDAVAndFilePublisher]], and [[pkg/application/application_test.go#TestSameNamedContactsPublishAndNotifyAdmin]].

### Recurrence and all-day parsing

Recurrences and exceptions must expand without treating RSVP status as assignment status. Invalid entries are reported while valid entries remain usable.

This intent is covered by [[pkg/parser/parser_test.go#TestRecurrenceExceptionsAndAttendeeStatuses]], [[pkg/parser/parser_test.go#TestCancelledOccurrenceIsNotAssigned]], and [[pkg/parser/parser_test.go#TestInvalidEntriesAreReportedWithoutDiscardingValidOnes]].

### Identifying invalid start dates

Invalid start-date notices identify the event by title, UID and start value, including timestamps with and without a timezone and missing or malformed values.

This is covered by [[pkg/parser/parser_test.go#TestInvalidStartIssueIdentifiesEvent]].

## Publication and mail safety

These tests ensure public output and messages contain only the intended data and fail safely.

### Escaped static page

The publisher escapes contact details, provides a telephone link and displays a UTF-8 unavailable message when no contact is available. A missing parent directory must prevent publication.

This intent is covered by [[pkg/publisher/publisher_test.go#TestPublishWritesEscapedStaticHTML]], [[pkg/publisher/publisher_test.go#TestPublishNilContactWritesUnavailableMessage]], and [[pkg/publisher/publisher_test.go#TestWebTemplateShowsCallLinkOrUnavailableMessage]].

### MIME and local submission

Messages must encode subjects safely, use UTF-8 text, reject header injection and propagate local mailer command failures. Delivery confirmation is intentionally limited to local submission.

This intent is covered by [[pkg/mailer/mailer_test.go#TestBuildMessageEncodesSubjectAndUsesUTF8]] and [[pkg/mailer/mailer_test.go#TestSystemMailerReturnsCommandFailure]].

## DAV transport and integration

Unit tests reject insecure DAV access and verify reads. Docker integration covers publication, reminders, refresh and failure notifications without production services.

The integration entry points are [[cmd/integration/scenarios.go#verifyInitialRun]] and [[cmd/integration/scenarios.go#verifyContactRefresh]], with infrastructure described in [[tests/integration/compose.yaml]].
