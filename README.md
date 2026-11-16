# Hilfmal

**«Hilfmal», der Pikettdienst für Jugendunruhen in Gemeinschaftsräumen.**

Ein freiwilliger Hilfsdienst für die Gemeinschaftsräume der WBGO. Mithelfen können alle Interessierten, Eltern sind besonders angesprochen.

Die diensthabende Person ist Anlaufstelle für Jugendliche und Anwohnende: Sie vermittelt, erinnert an die Regeln und hilft bei Bedarf, ein Treffen geordnet zu beenden, bevor eine Situation eskaliert.

## Idee in Kürze

- Freiwillige übernehmen den Dienst nach einem Dienstplan im Nextcloud-Kalender.
- Eine Webseite zeigt jederzeit Vorname und Telefonnummer der diensthabenden Person an, ein QR-Code im Gemeinschaftsraum führt dorthin.
- Automatische Erinnerungen unterstützen die Mitwirkenden und die Koordination.
- Nach einem Pilotjahr wird ausgewertet, ob sich das Modell bewährt.

## Dokumentation

- [Konzept](konzept.md): Idee, Organisation und Ziele des Pilotbetriebs.
- [Nutzerdokumentation](doc.md): Anleitung für Mitwirkende und Administration.
- [Architektur](architektur.md): Anforderungen, technische Entscheide und geplanter Betrieb.

## Status

Die PHP-Anwendung sowie schnelle und lokale Integrationstests sind vorhanden. Pipeline und Deployment sind noch nicht eingerichtet.

## Lokale Entwicklung

Voraussetzungen sind PHP 8.3 oder neuer, Composer und die im Composer-Manifest angegebenen PHP-Erweiterungen.

```bash
composer install
composer test
```

Für einen tatsächlichen Programmlauf `config.example.php` als Konfigurationsdatei ausserhalb des öffentlich erreichbaren Webverzeichnisses ablegen und die Werte anpassen. Das App-Passwort und die Notfall-E-Mail-Adresse müssen gesetzt sein. Die drei DAV-URLs bezeichnen den Kalender sowie die Adressbücher der Mitwirkenden und der Administration. Die Template-Pfade müssen auf die Dateien im Installationsverzeichnis zeigen; insbesondere sind sie beim Verschieben der Konfigurationsdatei anzupassen. Der Webserver muss die erzeugte `index.php` als PHP ausführen und der Programmlauf muss in das konfigurierte Webverzeichnis schreiben dürfen.

```bash
php bin/hilfmal.php /pfad/zur/konfiguration.php
```

Der Lauf verwendet die tatsächliche Systemzeit und liest die Daten über HTTPS. Er versendet echte E-Mails und veröffentlicht die Webseite: Für Entwicklung und Tests deshalb ausschliesslich isolierte Testkonten, Testempfänger und ein separates Ausgabeverzeichnis verwenden. Die Tests selbst kontaktieren keine externen Dienste.

Texte und Gestaltung stehen in `templates/`; die Pfade und E-Mail-Betreffzeilen sind konfigurierbar. Die Prüfspanne für Lücken ist ein PHP-Intervall wie `P1M`, der Erinnerungsvorlauf wird in Tagen angegeben. Der Zeitraum der Lückenprüfung beginnt am aktuellen Diensttag und endet vor dem Tag, der durch Addition des Intervalls entsteht; PHP-Kalendermonatsarithmetik gilt auch an Monatsenden.

### Integrationstests

Voraussetzungen sind ein laufender Docker-Daemon und Docker Compose v2. Der erste Lauf baut ein PHP-Image und lädt die festgelegten Images und Composer-Abhängigkeiten; dafür ist Internetzugriff nötig.

```bash
sh tests/integration/run.sh
```

Der Test startet eine eigene Nextcloud mit SQLite, ein TLS-Gateway, Mailpit und den PHP-Testcontainer in einem isolierten Docker-Netz ohne externe Verbindung oder veröffentlichte Ports. Ein Testaccount, Kalender, Adressbücher und synthetische Kontakte werden bei jedem Lauf neu erstellt. Der Mailversand über PHP `mail()` wird per `msmtp` an Mailpit umgeleitet; es werden keine echten E-Mails verschickt.

Geprüft werden DAV-Zugriff, Vertretung und Lücke, Erinnerung und Administrationshinweis, die über HTTP ausgelieferte Webseite, eine Datenänderung vor und nach einem erneuten Lauf, der reguläre Programmeinstieg und das Verhalten bei fehlgeschlagenem Datenzugriff und Ablauf der Anzeige. Die Anwendung wird dabei für die deterministischen Prüfläufe mit einer festen Uhrzeit aufgerufen; der reguläre Programmeinstieg verwendet wie im Betrieb die Systemzeit.

Das Skript verwendet einen eigenen Compose-Projektnamen und entfernt seine Container, das Netzwerk und die Testdaten auch bei einem Fehler. Es greift weder auf produktive Zugangsdaten noch auf eine bestehende Nextcloud oder Crontab zu. Die produzierten Docker-Images bleiben für spätere Läufe lokal erhalten.
