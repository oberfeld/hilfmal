# Hilfmal Knowledge Graph

Hilfmal operates a volunteer on-duty service for shared community rooms: it turns a Nextcloud plan into a public contact page and operational reminders.

This graph records the implemented current state and the rationale that is supported by project documentation or tests. It does not add deployment or operational intent that has not been implemented.

## Navigate

Use these anchors by concern: overview for purpose and pilot boundaries, domain for rules, architecture for runtime and safety, and tests for executable intent.

- [[overview]] — purpose, users, pilot goals, and the boundary of the service.
- [[domain]] — duty-day rules, overlap priority, contact resolution, and reporting.
- [[architecture]] — runtime flow, adapters, configuration, security, and failure behavior.
- [[tests]] — important test intent and executable coverage.

## Verified state

The Go application reads CalDAV and CardDAV data, calculates the duty, sends local-mail submissions, and atomically replaces a static HTML page. The CLI uses system time and a YAML configuration outside the public web directory.

The initial implementation and tests are present, but deployment and recurring Cron execution are not yet implemented. See [[architecture#Deployment status]] for the boundary and [[overview#Open review points]] for unknowns.
