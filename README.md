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

Die Go-Anwendung liegt als Modul direkt in der Repository-Wurzel; `cmd/` enthält die Programmeinstiege, `pkg/` die Fachpakete. Die GitHub Action führt die Go-Prüfungen und die isolierten Docker-Integrationstests mit Nextcloud und Mailpit aus. Deployment und Cronjob-Einrichtung sind noch nicht umgesetzt; die Anwendung ist daher noch nicht für ein produktives Deployment freigegeben.

Voraussetzungen sind eine installierte Go-Toolchain sowie Docker Compose v2 für die vollständige Verifikation.

```bash
make verify
```

`make verify` prüft die Formatierung, führt `go vet`, die Tests mit Race-Detector und Coverage aus, baut das Binary und startet die isolierten Nextcloud-/Mailpit-Integrationstests. Für nur die schnellen Go-Prüfungen können `make test`, `make vet` und `make build` getrennt ausgeführt werden.

Für einen Programmlauf `config.example.yaml` als YAML-Konfigurationsdatei ausserhalb des öffentlich erreichbaren Webverzeichnisses ablegen und die Werte anpassen. Das App-Passwort und die Notfall-E-Mail-Adresse müssen gesetzt sein. Die drei DAV-URLs bezeichnen den Kalender sowie die Adressbücher der Mitwirkenden und der Administration. Template-Pfade müssen auf die Dateien im Installationsverzeichnis zeigen.

```bash
make build
./bin/hilfmal /pfad/zur/konfiguration.yaml
```

Der Aufruf verwendet die Systemzeit und greift über HTTPS auf Nextcloud zu. Er versendet E-Mails über den lokalen `sendmail`-Befehl und veröffentlicht eine HTML-Datei. Deshalb nur isolierte Testkonten, Testempfänger und ein separates Ausgabeverzeichnis verwenden.

**Hinweis zur statischen Webseite:** Die HTML-Datei blendet Kontaktdaten nach einer festgelegten Zeit nicht automatisch aus. Wenn ein Lauf ausbleibt oder fehlschlägt, bleibt die zuletzt veröffentlichte Telefonnummer sichtbar, bis ein neuer erfolgreicher Lauf die Datei ersetzt.

Texte und Gestaltung stehen in `templates/`; die Pfade und E-Mail-Betreffzeilen sind konfigurierbar. `gap_interval` verwendet ein positives Datumsintervall wie `P1M`; der Erinnerungsvorlauf wird in Tagen angegeben.

## Tests und Migration

Die schnellen Go-Unit-Tests laufen ohne externe Dienste:

```bash
go test ./...
```

Die GitHub Action führt `make verify` aus. Der Integrationstest verwendet eine isolierte Nextcloud-Instanz und Mailpit; er benötigt keine produktiven Konten oder Zugangsdaten. Deployment und Cronjob-Einrichtung bleiben offen und müssen vor einer produktiven Umstellung auf Go umgesetzt und geprüft werden.
