# Overview

Hilfmal gives teenagers and neighbours a direct, low-visibility way to reach a present volunteer while keeping the volunteer role non-coercive and separate from supervision or policing.

The system supports the WBGO community-room Pikettdienst. A coordination person maintains Mitwirkende and a shared service calendar in Nextcloud. The application publishes only the first name and phone number of the person currently on duty, and sends operational email reminders.

## Purpose and users

The Pikettdienst is not a police or oversight role. Volunteers listen, reinforce rules and help end gatherings calmly; acute danger requires external help.

Three groups shape the domain:
- Volunteers cover a schedule and organize their own substitutions when unavailable.
- The coordination and administration maintain Nextcloud contacts and duty entries, and receive gaps or data problems.
- Neighbours and teenagers are readers of the public page or callers of the published number.

The product and organization goals are described in [[konzept.md]], while the participant workflow is in [[doc.md]].

## Service boundary

The application is a daily batch, not a live scheduling or publishing service. It reads data on each run, determines the current duty, creates email notices, and publishes a static HTML file.

Changes in Nextcloud appear only after the next successful run. A missed or failed run leaves the previous static page unchanged, so stale phone information is a consciously accepted operational risk. This behavior is implemented by [[pkg/application/application.go#Application#Run]] and [[pkg/publisher/publisher.go#FilePublisher#Publish]].

## Pilot goals

The pilot should review support and neighbourhood outcomes, volunteer sufficiency and coordination effort. These are operational measures rather than code metrics.

## Open review points

These points need operational or organizational confirmation before they can be described as verified project decisions.

- Production deployment and the exact Cyon Cron setup are documented as planned rather than implemented.
- The intended one-year pilot evaluation process and its success thresholds need validation by the WBGO.
- Confirmation of the local sendmail delivery behavior and the Cron host timezone belongs to a controlled Cyon pilot, not unit or Docker tests.
