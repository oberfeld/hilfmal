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

### Swiss service date and ranges

A run at 06:00 uses the new Swiss service date, while earlier times use the previous day. Calendar reads, month horizons and DST transitions must use the Swiss local date model consistently.

This intent is covered across [[pkg/planner/planner_test.go#TestServiceDate]], [[pkg/planner/planner_test.go#TestServiceDateAtDaylightSavingTransitions]], and [[pkg/application/application_test.go#TestMonthRangeUsesSwissLocalMidnight]].

## Calendar and contact resolution

The parser and planner tests protect the meaning of duties when entries overlap or recur.

### Priority and contact validity

Shorter events win overlapping entries, and equal-duration events use later creation time or UID as a deterministic final tie-break. Missing or ambiguous contacts are not fabricated; the next valid event may be selected and issues are reported.

This intent is covered by [[pkg/planner/planner_test.go#TestSelect]] and [[pkg/planner/planner_test.go#TestPriorityAndMissingCreationTieBreakAreInputOrderIndependent]].

### Recurrence and all-day parsing

Recurrences and exceptions must expand without treating RSVP status as assignment status. Invalid entries are reported while valid entries remain usable.

This intent is covered by [[pkg/parser/parser_test.go#TestRecurrenceExceptionsAndAttendeeStatuses]], [[pkg/parser/parser_test.go#TestCancelledOccurrenceIsNotAssigned]], and [[pkg/parser/parser_test.go#TestInvalidEntriesAreReportedWithoutDiscardingValidOnes]].

## Publication and mail safety

These tests ensure public output and messages contain only the intended data and fail safely.

### Escaped static page

The publisher must HTML-escape names and phone numbers and render an unavailable-service message instead of a false contact. A missing parent directory must prevent publication.

This intent is covered by [[pkg/publisher/publisher_test.go#TestPublishWritesEscapedStaticHTML]] and [[pkg/publisher/publisher_test.go#TestPublishNilContactWritesUnavailableMessage]].

### MIME and local submission

Messages must encode subjects safely, use UTF-8 text, reject header injection and propagate local mailer command failures. Delivery confirmation is intentionally limited to local submission.

This intent is covered by [[pkg/mailer/mailer_test.go#TestBuildMessageEncodesSubjectAndUsesUTF8]] and [[pkg/mailer/mailer_test.go#TestSystemMailerReturnsCommandFailure]].

## DAV transport and integration

Unit tests reject insecure DAV access and verify reads. Docker integration covers publication, reminders, refresh and failure notifications without production services.

The integration entry points are [[cmd/integration/scenarios.go#verifyInitialRun]] and [[cmd/integration/scenarios.go#verifyContactRefresh]], with infrastructure described in [[tests/integration/compose.yaml]].
