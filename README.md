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

Die PHP-Anwendung und schnelle automatisierte Tests werden umgesetzt. Integrationstestumgebung, Pipeline und Deployment sind noch nicht eingerichtet.

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
