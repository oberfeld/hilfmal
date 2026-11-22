# Hilfmal

![Hilfmal – Pikettdienst](public/hilfmal-logo.svg)

**«Hilfmal», der Pikettdienst für Jugendunruhen in Gemeinschaftsräumen.**

Ein freiwilliger Hilfsdienst für die Gemeinschaftsräume der WBGO. Mithelfen können alle Interessierten, Eltern sind besonders angesprochen.

Die diensthabende Person ist Anlaufstelle für Jugendliche und Anwohnende: Sie vermittelt, erinnert an die Regeln und hilft bei Bedarf, ein Treffen geordnet zu beenden, bevor eine Situation eskaliert.

## Idee in Kürze

- Freiwillige übernehmen den Dienst nach einem Dienstplan im Nextcloud-Kalender.
- Eine Webseite zeigt Vorname und Telefonnummer der diensthabenden Person an.
- Automatische Erinnerungen unterstützen die Mitwirkenden und die Koordination.

## Dokumentation

- [Konzept](konzept.md): organisatorische Idee und Ziele des Pilotbetriebs.
- [Nutzerdokumentation](doc.md): praktische Anleitung für Mitwirkende und Administration.
- [Architektur](architektur.md): übergeordnete technische Entscheide und offene Betriebsprüfungen.
- [lat.md/](lat.md/): code-nahe Beschreibung des implementierten Verhaltens, der Fachregeln und Tests.

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

Texte und Gestaltung stehen in `templates/`, öffentliche statische Dateien in `public/`; Pfade und E-Mail-Betreffzeilen sind konfigurierbar.

## Tests

Die Testfälle und ihr fachlicher Zweck sind in [lat.md/tests.md](lat.md/tests.md) dokumentiert. Die schnellen Tests benötigen keine externen Dienste:

```bash
make test-application
go test ./...
```

Die Kontaktzuordnungs-Szenarien lassen sich separat ausführen. Mutationstests für alle Go-Pakete sind nicht Teil von `make verify` und dauern deutlich länger:

```bash
go test -v ./pkg/planner -run TestContactFeatures
make mutation-test
```

Die GitHub Action führt `make verify` aus. Darin enthalten sind auch isolierte Docker-Integrationstests mit Nextcloud und Mailpit; produktive Konten oder Zugangsdaten werden dafür nicht verwendet.

### Cyon-Deployment

Pushes auf `main` deployen das statisch gelinkte Linux-Binary (ohne glibc-Abhängigkeit), die Templates und `config.cyon.yaml` nach `/home/wohnenim/hilfmal`. Anschliessend aktualisiert `hilfmal --publish-only config.yaml` die Webseite, ohne Erinnerungen, Administrationshinweise oder Fehler-E-Mails zu versenden. Der Modus verwendet dieselbe Konfiguration und dieselben Templates wie der vollständige Lauf. Fehler werden protokolliert und lassen das Deployment fehlschlagen. Die erzeugte Webseite liegt unter `/home/wohnenim/public_html/hilfmal/public/index.html`. Die Action setzt den Cyon-Cronjob anhand von `cron_schedule` in `config.cyon.yaml`; dieser führt den vollständigen Programmlauf täglich um 06:00 Uhr in der Zeitzone des Cyon-Servers aus.

Alle Dateien unter `public/` werden in das öffentliche Unterverzeichnis `assets/` kopiert (Dateien mit Rechten `644`, Verzeichnisse mit `755`). Bereits vorhandene Dateien wie die erzeugte `index.html` bleiben erhalten.

Unter **Settings → Secrets and variables → Actions** müssen eingerichtet sein:

- Secret `PRIVATE_KEY`: der SSH-Private-Key für Cyon. Ein im Chat offengelegter Schlüssel muss zuerst rotiert und auf Cyon ersetzt werden.
- Secrets `CHISCHTE_UN` und `CHISCHTE_PW`: Nextcloud-Benutzername und -Passwort. Die Action schreibt sie als `HILFMAL_DAV_USER` und `HILFMAL_DAV_PASSWORD` in `/home/wohnenim/hilfmal/.env`; der Cronjob lädt die Datei.
- Variable `CYON_KNOWN_HOSTS`: die verifizierte Known-Hosts-Zeile für exakt `oberfeld.be` (die `HostName`-Angabe in der Action), damit die Action den Server authentifiziert. Bei `ssh-keyscan -H` nur die Zeile ab `|1|...` übernehmen; Kommentarzeilen mit `#` sind optional.

Kalender-/Adressbuch-URLs sowie Absender- und Notfall-E-Mail-Adresse müssen in `config.cyon.yaml` eingetragen werden. Die Pfade und übrigen festen Einstellungen für Cyon stehen ebenfalls dort.
Für die Prüfung mit `make config-check` müssen `calendar_url`, `contacts_url`, `admins_url`, `sender`, `emergency_email`, `sendmail_path` und `cron_schedule` jeweils genau einmal ohne Einrückung als nichtleerer, doppelt zitierter Wert in einer Zeile stehen. `sendmail_path` muss absolut sein. Der Cron-Ausdruck besteht aus fünf numerischen Feldern mit den Zeichen `*`, `,`, `-` und `/`.
