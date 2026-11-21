# Architecture

Hilfmal is a Go command that orchestrates four replaceable concerns: DAV data access, planning, email submission and static-page publication.

## Runtime flow

The CLI loads a YAML configuration, validates required secrets and paths, creates the Nextcloud and local-mail adapters, and invokes [[pkg/application/application.go#Application#Run]] once with the system time.

Each run:
1. loads administration recipients and records a stage-specific failure if that address book is unavailable;
2. converts the run time to the Swiss service date and computes reminder and gap horizons;
3. reads the relevant calendar and Mitwirkende contacts;
4. selects the current and reminder contacts and calculates coverage issues;
5. sends reminder and administration mail when applicable;
6. publishes a static page even after mail or administrative-read failures, but reports failure and tries failure notifications.

This separation makes a data adapter failure, mail failure and publication failure observable through exit status while preserving the publication fallback used for contact and mail-stage errors.

The deployment uses [[pkg/application/application.go#Application#PublishOnly]] on the same configured application to refresh only the current page. This mode does not load administrator contacts or send mail; failures are logged and returned to the deployment.

## Data adapters

[[pkg/datasource/interface.go#DataSource]] defines Admins, Contacts, Events and a list of source issues. [[pkg/datasource/nextcloud.go#NextcloudSource]] implements it through CalDAV and CardDAV REPORT requests over HTTPS.

The HTTP DAV transport requires HTTPS, sets Basic Authentication, rejects non-Multi-Status responses and limits DAV response data. Source issues collected from calendar parsing are retained for the application to include in administration mail.

## Configuration and templates

The `models.Config` shape in [[pkg/models/models.go#Config]] is the configuration contract. [[config.cyon.yaml]] documents DAV URLs, reminder lead time, gap interval and file paths; credentials are supplied separately.

Templates separate copy and presentation from logic: [[templates/reminder.txt]], [[templates/admin.txt]], and [[templates/web.html]]. Sender and subject values are configuration, and the local mail implementation in [[pkg/mailer/mailer.go#SystemMailer]] submits UTF-8 MIME messages to a sendmail-compatible command.

## Failure and security behavior

Mail, publication and DAV failures are logged and produce exit status 1. A mail send that is accepted by the local mailer only confirms local submission, not final delivery.

Administrative recipients are de-duplicated and complemented by the emergency address for failure reports. Configuration and secrets must remain outside the public web directory, and DAV endpoints must use HTTPS. The application never exposes a test clock or operational control endpoint.

## Deployment status

The GitHub workflow deploys a statically linked Linux binary and templates to Cyon and configures a daily cron run at 06:00 server time. The binary does not depend on the server's glibc version.

The Cyon YAML configuration is checked by [[Makefile]] during verification, while credentials are supplied by GitHub secrets and stored outside the public directory.
