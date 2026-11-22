# Hilfmal

**«Hilfmal», der Pikettdienst für Jugendunruhen in Gemeinschaftsräumen.**

Ein freiwilliger Hilfsdienst für die Gemeinschaftsräume der WBGO. Mithelfen können alle Interessierten, Eltern sind besonders angesprochen.

Die diensthabende Person ist Anlaufstelle für Jugendliche und Anwohnende: Sie vermittelt, erinnert an die Regeln und hilft bei Bedarf, ein Treffen geordnet zu beenden, bevor eine Situation eskaliert.

## Idee in Kürze

- Freiwillige übernehmen den Dienst nach einem Dienstplan im Nextcloud-Kalender.
- Eine Webseite zeigt Vorname und Telefonnummer der diensthabenden Person an.
- Automatische Erinnerungen unterstützen die Mitwirkenden und die Koordination.

## Dokumentation

- [Konzept](konzept.md): Idee, Organisation und Ziele des Pilotbetriebs.
- [Nutzerdokumentation](doc.md): Anleitung für Mitwirkende und Administration.
- [Architektur](architektur.md): technische Anforderungen und aktuelle Entscheide.

## Go-Anwendung

Die Go-Anwendung liegt als Modul direkt in der Repository-Wurzel; `cmd/` enthält die Programmeinstiege, `pkg/` die Fachpakete. Die GitHub Action führt die Go-Prüfungen und die isolierten Docker-Integrationstests mit Nextcloud und Mailpit aus. Pushes auf `main` deployen die Linux-Anwendung zu Cyon und aktualisieren den täglichen Cronjob.

Voraussetzungen sind eine installierte Go-Toolchain sowie Docker Compose v2 für die vollständige Verifikation.

```bash
make verify
```

`make verify` prüft die Cyon-Konfiguration und Formatierung, führt `go vet`, die Tests mit Race-Detector und Coverage aus, baut das Binary und startet die isolierten Nextcloud-/Mailpit-Integrationstests. Für nur die schnellen Prüfungen können `make config-check`, `make test`, `make vet` und `make build` getrennt ausgeführt werden.

Für einen lokalen Programmlauf `config.cyon.yaml` als `config.local.yaml` kopieren und die Werte für eine isolierte Testumgebung anpassen. Insbesondere müssen die DAV-URLs, E-Mail-Adressen sowie Template- und Ausgabepfade angepasst werden. Nextcloud-Benutzername und App-Passwort können in der lokalen YAML-Datei als `dav_user` und `dav_password` oder über `HILFMAL_DAV_USER` und `HILFMAL_DAV_PASSWORD` gesetzt werden. Zugangsdaten und lokale Konfiguration ausserhalb des öffentlichen Webverzeichnisses aufbewahren und nicht einchecken.

```bash
make build
./bin/hilfmal config.local.yaml
```

Der Aufruf verwendet die Systemzeit und greift über HTTPS auf Nextcloud zu. Er versendet E-Mails über den unter `sendmail_path` konfigurierten absoluten Pfad und veröffentlicht eine HTML-Datei. Ohne diese Einstellung wird aus Kompatibilitätsgründen `sendmail` im `PATH` gesucht. Deshalb nur isolierte Testkonten, Testempfänger und ein separates Ausgabeverzeichnis verwenden.

**Hinweis zur statischen Webseite:** Die HTML-Datei blendet Kontaktdaten nach einer festgelegten Zeit nicht automatisch aus. Wenn ein Lauf ausbleibt oder fehlschlägt, bleibt die zuletzt veröffentlichte Telefonnummer sichtbar, bis ein neuer erfolgreicher Lauf die Datei ersetzt.

Texte und Gestaltung stehen in `templates/`; die Pfade und E-Mail-Betreffzeilen sind konfigurierbar. Das Web-Template unterstützt `{{name}}`, `{{phone}}` und `{{phone_url}}` für den Anruflink. `{{contact_hidden}}` und `{{unavailable_hidden}}` blenden je nach Dienstinformation den entsprechenden Bereich aus. `gap_interval` verwendet ein positives Datumsintervall wie `P1M`; der Erinnerungsvorlauf wird in Tagen angegeben.

## Tests und Migration

Die Application-Szenarien sind als Godog-BDD formuliert. Sie testen den vollständigen Programmlauf mit In-Memory-Adaptern – einschliesslich fehlgeschlagenem E-Mail-Versand, nicht lesbarem Admin-Adressbuch und trotzdem ausgeführter Veröffentlichung – ohne Nextcloud, SMTP oder Docker. Der schnelle, lesbare Lauf zeigt die geprüften Szenarien direkt an:

```bash
make test-application
```

Die Szenarien zur Kontaktzuordnung stehen in `pkg/planner/features/kontaktzuordnung.feature` und lassen sich mit `go test -v ./pkg/planner -run TestContactFeatures` ausführen.

Die übrigen schnellen Go-Tests laufen ebenfalls ohne externe Dienste:

```bash
go test ./...
```

Mutationstests für alle Go-Pakete laufen separat und sind nicht Teil von `make verify`, da sie deutlich länger dauern können. Gremlins wird in der festgelegten Version beim ersten Aufruf automatisch geladen:

```bash
make mutation-test
```

Die GitHub Action führt `make verify` aus. Der davon getrennte Integrationstest verwendet eine isolierte Nextcloud-Instanz und Mailpit; er benötigt keine produktiven Konten oder Zugangsdaten.

### Cyon-Deployment

Pushes auf `main` deployen das statisch gelinkte Linux-Binary (ohne glibc-Abhängigkeit), die Templates und `config.cyon.yaml` nach `/home/wohnenim/hilfmal`. Anschliessend aktualisiert `hilfmal --publish-only config.yaml` die Webseite, ohne Erinnerungen, Administrationshinweise oder Fehler-E-Mails zu versenden. Der Modus verwendet dieselbe Konfiguration und dieselben Templates wie der vollständige Lauf. Fehler werden protokolliert und lassen das Deployment fehlschlagen. Die erzeugte Webseite liegt unter `/home/wohnenim/public_html/hilfmal/public/index.html`. Die Action setzt den Cyon-Cronjob anhand von `cron_schedule` in `config.cyon.yaml`; dieser führt den vollständigen Programmlauf in der Zeitzone des Cyon-Servers aus. Für den Test ist vorübergehend täglich 09:30 eingestellt; danach `cron_schedule` wieder auf `0 6 * * *` setzen und deployen.

Unter **Settings → Secrets and variables → Actions** müssen eingerichtet sein:

- Secret `PRIVATE_KEY`: der SSH-Private-Key für Cyon. Ein im Chat offengelegter Schlüssel muss zuerst rotiert und auf Cyon ersetzt werden.
- Secrets `CHISCHTE_UN` und `CHISCHTE_PW`: Nextcloud-Benutzername und -Passwort. Die Action schreibt sie als `HILFMAL_DAV_USER` und `HILFMAL_DAV_PASSWORD` in `/home/wohnenim/hilfmal/.env`; der Cronjob lädt die Datei.
- Variable `CYON_KNOWN_HOSTS`: die verifizierte Known-Hosts-Zeile für exakt `oberfeld.be` (die `HostName`-Angabe in der Action), damit die Action den Server authentifiziert. Bei `ssh-keyscan -H` nur die Zeile ab `|1|...` übernehmen; Kommentarzeilen mit `#` sind optional.

Kalender-/Adressbuch-URLs sowie Absender- und Notfall-E-Mail-Adresse müssen in `config.cyon.yaml` eingetragen werden. Die Pfade und übrigen festen Einstellungen für Cyon stehen ebenfalls dort.
Für die Prüfung mit `make config-check` müssen `calendar_url`, `contacts_url`, `admins_url`, `sender`, `emergency_email`, `sendmail_path` und `cron_schedule` jeweils genau einmal ohne Einrückung als nichtleerer, doppelt zitierter Wert in einer Zeile stehen. `sendmail_path` muss absolut sein. Der Cron-Ausdruck besteht aus fünf numerischen Feldern mit den Zeichen `*`, `,`, `-` und `/`.
