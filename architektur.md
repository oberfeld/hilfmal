# Architektur von Hilfmal

## Überblick

In diesem Repository wird das PHP-Programm für die zentrale Logik von Hilfmal entwickelt. Der Dienstplan und das Adressbuch bleiben in Nextcloud; das Programm läuft auf dem Cyon-Server. Eine GitHub Action installiert das Programm bei jedem Push auf den Branch `main` und richtet dessen täglichen Aufruf ein.

Die fachlichen Grundlagen stehen im [Konzept](konzept.md); die Bedienung für Mitwirkende und Administration ist in der [Nutzerdokumentation](doc.md) beschrieben. Die nachfolgenden Anforderungen beschreiben die geplante Umsetzung.

## Daten und Zugriff

- Der Dienstplan wird über CalDAV aus dem Nextcloud-Kalender gelesen.
- Die Mitwirkenden und die Administrationsadressen werden über CardDAV aus zwei getrennten Nextcloud-Adressbüchern gelesen.
- Die eingeladene Person eines Kalendereintrags wird anhand ihrer eindeutigen E-Mail-Adresse einem Kontakt zugeordnet. Vorname und Telefonnummer stammen aus diesem Kontakt.
- Der Einladungsstatus wird nicht berücksichtigt: Ob eine Person zugesagt, abgesagt oder noch nicht geantwortet hat, ändert die Zuordnung nicht.
- Für den Zugriff auf Kalender und Adressbücher wird für den Service-Account ein eigenes App-Passwort erzeugt und verwendet.

Der eigene Nextcloud-Service-Account besitzt den Kalender und beide Adressbücher, hat aber keine Nextcloud-Adminrechte. Die Koordination pflegt die Daten über ihren persönlichen Account und entsprechende Freigaben. Das App-Passwort hat die Rechte des Service-Accounts und ist entsprechend zu schützen.

### Entscheid zur Personenzuordnung

Die Kontakte im Adressbuch «Mitwirkende» enthalten Vorname, Telefonnummer und eine für die Zuordnung eindeutige E-Mail-Adresse. Die diensthabende Person wird einzeln als Teilnehmerin oder Teilnehmer zum Kalendereintrag hinzugefügt; ihre E-Mail-Adresse verbindet den Eintrag mit dem Kontakt.

Als Alternativen wurden Nextcloud Tables mit einer Personenkennung im Kalendereintrag sowie eine Tabellen- oder JSON-Datei verworfen. Sie ermöglichen ebenfalls das Auslesen durch das Programm, erfordern aber eine separat gepflegte Zuordnung und nutzen die Teilnehmerfunktion des Kalenders nicht direkt.

## Dienstzeiten und Auswahl

- Ein Dienst ist lückenlos an jedem Tag erforderlich. Die Kalendereinträge werden als ganztägige Einträge erfasst. Ein Eintrag für einen Tag gilt als Dienst von **06:00 Uhr dieses Tages bis 06:00 Uhr des nächsten Tages**, jeweils in Schweizer Zeit (`Europe/Zurich`).
- Auch ein mehrtägiger ganztägiger Eintrag kann mehrere aufeinanderfolgende Dienste abdecken. Die Dauer des gesamten Eintrags wird für die Auswahl bei Überschneidungen verwendet.
- Überschneiden sich Einträge, hat der kürzere Eintrag Vorrang. Sind sie gleich lang, hat der später erstellte Eintrag Vorrang. So kann eine Vertretung durch einen kürzeren oder bei gleicher Dauer später erstellten Eintrag organisiert werden.
- Pro Dienstzeitraum wird wenn möglich genau ein Kontakt ausgewählt und auf der Webseite angezeigt. Kann anhand der Angaben keine eindeutige Person bestimmt werden, wird keine falsche Telefonnummer veröffentlicht und die Administration informiert.

## Aufgaben des PHP-Programms

Der tägliche Programmlauf liest und prüft den Dienstplan und die Adressbücher. Benachrichtigungen werden ausschliesslich per E-Mail verschickt. Er übernimmt folgende Aufgaben:

1. **Mitwirkende benachrichtigen:** Die eingeteilte Person wird an ihren bevorstehenden Dienst und an die Organisation einer Vertretung bei Verhinderung erinnert.
2. **Lücken melden:** Die Administration erhält einen Hinweis auf Tage ohne eingeteilten Dienst. Eine lückenlose Abdeckung ist jeden Tag erforderlich.
3. **Unvollständige Einträge melden:** Die Administration erhält einen Hinweis auf Dienstplaneinträge, die für die Verarbeitung erforderliche Angaben vermissen oder keine eindeutige Zuordnung zu einem Kontakt erlauben. Dazu gehören insbesondere fehlende Datumsangaben, eine fehlende oder mehrdeutige eingeladene Person sowie fehlende oder mehrdeutige Kontaktdaten.
4. **Webseite bereitstellen:** Das Programm stellt die Webseite mit Vorname und Telefonnummer der aktuell diensthabenden Person bereit. Andere Angaben aus Kalender und Adressbuch werden nicht öffentlich angezeigt.

Für E-Mails und Webseite stehen einfach anzupassende Templates bereit. Diese trennen die Gestaltung und Texte von der Programmlogik.

Der tägliche Lauf aktualisiert die Daten und die Webseite um 06:00 Uhr. Zwischen zwei Läufen werden Änderungen in Nextcloud nicht automatisch übernommen; sie erscheinen erst nach dem nächsten erfolgreichen Lauf. Die bis zum Abschluss des Laufs angezeigte Information kann deshalb noch den vorherigen Dienst zeigen.

Wenn keine Person eindeutig zugeordnet werden kann oder die Daten nicht verlässlich aktuell sind, darf keine möglicherweise falsche Telefonnummer angezeigt werden. Stattdessen zeigt die Webseite an, dass keine verlässliche Dienstinformation verfügbar ist. Fehler beim Abrufen oder Verarbeiten der Daten werden protokolliert und der Administration gemeldet; sie dürfen nicht als erfolgreicher Lauf erscheinen. Die Aufbewahrung der Daten und Protokolle ist auf das für den Betrieb Nötige zu begrenzen.

## Installation und Zeitsteuerung

- Ein Push auf `main` startet die GitHub Action. Sie installiert die aktuelle Version des Programms auf dem Cyon-Server.
- Dieselbe Action richtet auf dem Cyon-Server einen Cronjob ein, der das Programm täglich um **06:00 Uhr Schweizer Zeit** (`Europe/Zurich`, einschliesslich Sommer- und Winterzeit) ausführt. Die Zeitzone muss bei der Einrichtung und im Betrieb ausdrücklich berücksichtigt werden.
- Die Einrichtung ist wiederholbar: Ein erneutes Deployment aktualisiert den zu Hilfmal gehörenden Cronjob, ohne doppelte Einträge anzulegen oder andere Cronjobs zu verändern.
- Ein fehlgeschlagenes Deployment oder eine fehlgeschlagene Cronjob-Einrichtung lässt die Action fehlschlagen. Auch der regelmässige Programmlauf muss auf Fehler überwacht werden; ein erfolgreiches Deployment allein bestätigt nicht, dass die täglichen Läufe funktionieren.
- Als Vorlage dient der Deployment-Workflow aus dem benachbarten Projekt Putzmal (`.github/workflows/php.yml`): GitHub Actions richtet PHP ein, installiert Composer-Abhängigkeiten und überträgt das Programm per SSH und `rsync` auf den Server. Anschliessend wird der Cronjob per SSH eingerichtet. Die Übertragung ist für Hilfmal so anzupassen, dass Konfiguration und andere Daten nicht unbeabsichtigt gelöscht werden und der SSH-Hostschlüssel geprüft wird.

## Zugangsdaten und Sicherheit

Das App-Passwort für den Zugriff auf Adressbücher und Kalender wird als GitHub-Actions-Secret hinterlegt und weder im Repository noch in Workflow-Ausgaben veröffentlicht. Auch die für die Installation auf Cyon benötigten Zugangsdaten sind getrennt und geschützt zu verwalten.

Da der Cronjob unabhängig von GitHub auf dem Server läuft, benötigt das Programm die Nextcloud-Zugangsdaten auch dort zur Laufzeit. Die Action stellt sie beim Deployment in einer geschützten Serverkonfiguration ausserhalb des öffentlichen Webverzeichnisses bereit. Nur der für das Programm benötigte Server-Account darf sie lesen. Zugangsdaten und personenbezogene Daten dürfen nicht in öffentlich ausgelieferten Dateien, Logs oder Fehlermeldungen erscheinen. Bei einem Passwortwechsel müssen die Secrets und die Serverkonfiguration aktualisiert und der Zugriff erneut geprüft werden.

## Vor der Umsetzung festzulegen

- Absender und technische Einrichtung des E-Mail-Versands.
- Wie weit im Voraus Mitwirkende erinnert werden und welcher Zeitraum auf Lücken geprüft wird.
- Wie bei gleich langen Einträgen mit identischem oder fehlendem Erstellungszeitpunkt entschieden wird.
- Wie lange die zuletzt eingelesenen Daten für die Webseite als verlässlich aktuell gelten und wie Fehler der täglichen Läufe an die Administration gemeldet werden.
- Die konkreten Serverpfade und die Cronjob-Einrichtung auf dem gewählten Cyon-Angebot.
