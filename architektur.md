# Architektur und Betriebsentscheide

## Geltungsbereich

Dieses Dokument hält die übergeordneten technischen Entscheide und noch nötigen Betriebsprüfungen fest. Die detaillierte Beschreibung des implementierten Verhaltens, der Fachregeln und der Tests mit Verweisen auf den Code liegt im [lat-Wissensgraphen](lat.md/).

Das [Konzept](konzept.md) beschreibt Zweck und Pilotziele; die [Nutzerdokumentation](doc.md) erklärt die Abläufe für Mitwirkende und Administration.

## Technische Entscheide

- Nextcloud bleibt die führende Quelle für Dienstplan und Kontaktdaten. Das Programm führt keine parallele Stammdatenpflege ein.
- Hilfmal läuft als täglicher Batch. Die Webseite ist statisches HTML und enthält öffentlich nur Vorname und Telefonnummer der diensthabenden Person.
- E-Mails werden über den lokalen sendmail-kompatiblen Dienst des Servers übergeben. Das Programm verwaltet keine eigenen SMTP-Zugangsdaten.
- Deployment und Cronjob werden über GitHub Actions eingerichtet. Die Zugangsdaten werden für die Laufzeit ausserhalb des öffentlichen Webverzeichnisses auf dem Server bereitgestellt.
- Änderungen in Nextcloud werden erst beim nächsten erfolgreichen Lauf veröffentlicht. Bei einem verpassten oder fehlgeschlagenen Lauf bleibt die zuletzt veröffentlichte Telefonnummer sichtbar; diese Einschränkung muss im Betrieb berücksichtigt werden.

Die genauen DAV-Verträge, Zuordnungs- und Prioritätsregeln, Fehlerbehandlung und Testabdeckung stehen in [lat.md/architecture.md](lat.md/architecture.md), [lat.md/domain.md](lat.md/domain.md) und [lat.md/tests.md](lat.md/tests.md).

## Vor dem Pilotbetrieb zu prüfen

Die organisatorische Auswertung nach dem Pilotjahr und ihre Erfolgskriterien müssen durch die WBGO festgelegt werden; siehe [Konzept](konzept.md).
