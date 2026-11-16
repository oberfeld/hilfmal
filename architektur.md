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
- Kann die Priorität damit nicht bestimmt werden, etwa bei gleichem oder fehlendem Erstellungszeitpunkt, werden die UUIDs der Einträge alphanumerisch aufsteigend sortiert. Der erste Eintrag hat Vorrang.
- Fehlt bei mindestens einem Eintrag gleicher Dauer der Erstellungszeitpunkt, entscheidet für diese Dauergruppe die UUID-Reihenfolge. So bleibt die Auswahl unabhängig von der Reihenfolge der eingelesenen Einträge.
- Pro Dienstzeitraum wird wenn möglich genau ein Kontakt ausgewählt und auf der Webseite angezeigt. Ist der Kontakt des vorrangigen Eintrags nicht eindeutig oder unvollständig, wird der nächste gültige Eintrag verwendet und die Administration informiert. Gibt es keinen gültigen Eintrag, wird keine Telefonnummer veröffentlicht.
- Wiederkehrende ganztägige Einträge und deren Ausnahmen werden berücksichtigt.

## Aufgaben des PHP-Programms

Der tägliche Programmlauf liest und prüft den Dienstplan und die Adressbücher. Benachrichtigungen werden ausschliesslich per E-Mail verschickt. Er übernimmt folgende Aufgaben:

1. **Mitwirkende benachrichtigen:** Die eingeteilte Person wird einmal vor Beginn eines zusammenhängenden Dienstes an ihren bevorstehenden Dienst und an die Organisation einer Vertretung bei Verhinderung erinnert. Der Vorlauf beträgt standardmässig sieben Tage und ist konfigurierbar.
2. **Lücken melden:** Die Administration erhält einen Hinweis auf Tage ohne eingeteilten Dienst. Eine lückenlose Abdeckung ist jeden Tag erforderlich. Geprüft wird standardmässig ein Monat in die Zukunft; dieser Zeitraum ist konfigurierbar.
3. **Unvollständige Einträge melden:** Die Administration erhält einen Hinweis auf Dienstplaneinträge, die für die Verarbeitung erforderliche Angaben vermissen oder keine eindeutige Zuordnung zu einem Kontakt erlauben. Dazu gehören insbesondere fehlende Datumsangaben, eine fehlende oder mehrdeutige eingeladene Person sowie fehlende oder mehrdeutige Kontaktdaten.
4. **Webseite bereitstellen:** Das Programm stellt die Webseite mit Vorname und Telefonnummer der aktuell diensthabenden Person bereit. Andere Angaben aus Kalender und Adressbuch werden nicht öffentlich angezeigt.

Für E-Mails und Webseite stehen einfach anzupassende Templates bereit. Diese trennen die Gestaltung und Texte von der Programmlogik. Als Absenderadresse wird standardmässig `hilfmal@oberfeld.be` verwendet; sie ist konfigurierbar.

### E-Mail-Versand

Wie Putzmal verwendet Hilfmal die PHP-Funktion `mail()` und den auf dem Cyon-Server eingerichteten Mailversand, ohne eigene SMTP-Zugangsdaten im Programm. Die konfigurierte Absenderadresse wird für `From` und `Reply-To` verwendet. Texte werden als UTF-8 versendet; Betreff und Header müssen für den Mailversand korrekt kodiert werden.

Der Rückgabewert von `mail()` wird geprüft. Ein Fehler beim Versand wird protokolliert und der Programmlauf als fehlgeschlagen ausgewiesen. Anders als bei Putzmal darf ein fehlgeschlagener Einzelversand nicht zu einem erfolgreichen Lauf führen. Ein erfolgreicher Rückgabewert bestätigt nur die Annahme durch den lokalen Mailversand, nicht die Zustellung an die Empfänger.

Der tägliche Lauf liest die Daten jeweils neu und aktualisiert die Webseite um 06:00 Uhr. Es wird kein Cache der eingelesenen Kalender- oder Adressbuchdaten gepflegt. Das veröffentlichte Ergebnis gilt bis zum nächsten vorgesehenen Cronjob. Zwischen zwei Läufen werden Änderungen in Nextcloud nicht automatisch übernommen; sie erscheinen erst nach dem nächsten erfolgreichen Lauf.

Wenn keine Person eindeutig zugeordnet werden kann oder die Gültigkeit des veröffentlichten Ergebnisses abgelaufen ist, darf keine möglicherweise falsche Telefonnummer angezeigt werden. Stattdessen zeigt die Webseite an, dass keine verlässliche Dienstinformation verfügbar ist. Dies gilt auch, wenn der nächste Lauf ausbleibt oder fehlschlägt. Die technische Umsetzung muss die Gültigkeit deshalb auch ohne erfolgreichen Cronjob beim Abruf durchsetzen. Fehler beim Abrufen, Verarbeiten oder Veröffentlichen werden protokolliert und der Administration per E-Mail gemeldet; sie dürfen nicht als erfolgreicher Lauf erscheinen. Falls der E-Mail-Versand selbst fehlschlägt, wird auch dieser Fehler protokolliert und der Lauf als fehlgeschlagen ausgewiesen. Die Aufbewahrung der Daten und Protokolle ist auf das für den Betrieb Nötige zu begrenzen.

Für den Fall, dass das Administrationsadressbuch nicht gelesen werden kann, ist zusätzlich eine Notfall-E-Mail-Adresse konfigurierbar. Fehler beim Programmstart können nur per E-Mail gemeldet werden, wenn Konfiguration, Absender und Notfalladresse bereits verfügbar sind; andernfalls bleiben Protokoll und fehlgeschlagener Exit-Status.

## Installation und Zeitsteuerung

- Ein Push auf `main` startet die GitHub Action. Sie installiert die aktuelle Version des Programms auf dem Cyon-Server.
- Dieselbe Action richtet auf dem Cyon-Server einen Cronjob ein, der das Programm täglich um **06:00 Uhr Schweizer Zeit** (`Europe/Zurich`, einschliesslich Sommer- und Winterzeit) ausführt. Die Zeitzone muss bei der Einrichtung und im Betrieb ausdrücklich berücksichtigt werden.
- Die Einrichtung ist wiederholbar: Ein erneutes Deployment aktualisiert den zu Hilfmal gehörenden Cronjob, ohne doppelte Einträge anzulegen oder andere Cronjobs zu verändern.
- Ein fehlgeschlagenes Deployment oder eine fehlgeschlagene Cronjob-Einrichtung lässt die Action fehlschlagen. Auch der regelmässige Programmlauf muss auf Fehler überwacht werden; ein erfolgreiches Deployment allein bestätigt nicht, dass die täglichen Läufe funktionieren.
- Als Vorlage dient der Deployment-Workflow aus dem benachbarten Projekt Putzmal (`.github/workflows/php.yml`): GitHub Actions richtet PHP ein, installiert Composer-Abhängigkeiten und überträgt das Programm per SSH und `rsync` auf den Server. Anschliessend wird der Cronjob per SSH eingerichtet. Die Übertragung ist für Hilfmal so anzupassen, dass Konfiguration und andere Daten nicht unbeabsichtigt gelöscht werden und der SSH-Hostschlüssel geprüft wird.
- Serverpfade, insbesondere für Installation, geschützte Konfiguration und veröffentlichte Webseite, müssen einfach konfigurierbar sein; Deployment und Cronjob verwenden die entsprechend konfigurierten Pfade.

Wie bei Putzmal liest die Action die bestehende Crontab über SSH, aktualisiert den Hilfmal-Eintrag und installiert die angepasste Crontab mit `crontab`. Der Eintrag wechselt in das konfigurierte Installationsverzeichnis und startet das PHP-Programm über die Kommandozeile. Für Hilfmal wird der Zeitplan auf täglich 06:00 Uhr Schweizer Zeit gesetzt; vor Inbetriebnahme ist sicherzustellen, dass die Zeitzone des Cron-Dienstes dies auch bei Sommer- und Winterzeit gewährleistet. Anders als beim Putzmal-Beispiel darf ein Fehler beim Lesen einer bestehenden Crontab nicht dazu führen, dass fremde Einträge überschrieben werden.

## Zugangsdaten und Sicherheit

Das App-Passwort für den Zugriff auf Adressbücher und Kalender wird als GitHub-Actions-Secret hinterlegt und weder im Repository noch in Workflow-Ausgaben veröffentlicht. Auch die für die Installation auf Cyon benötigten Zugangsdaten sind getrennt und geschützt zu verwalten.

Da der Cronjob unabhängig von GitHub auf dem Server läuft, benötigt das Programm die Nextcloud-Zugangsdaten auch dort zur Laufzeit. Die Action stellt sie beim Deployment in einer geschützten Serverkonfiguration ausserhalb des öffentlichen Webverzeichnisses bereit. Nur der für das Programm benötigte Server-Account darf sie lesen. Zugangsdaten und personenbezogene Daten dürfen nicht in öffentlich ausgelieferten Dateien, Logs oder Fehlermeldungen erscheinen. Bei einem Passwortwechsel müssen die Secrets und die Serverkonfiguration aktualisiert und der Zugriff erneut geprüft werden.

Vor Inbetriebnahme sind der lokale Mailversand, die Cron-Zeitzone und die Berechtigung zur Einrichtung der Crontab auf dem gewählten Cyon-Angebot zu prüfen.

## Testkonzept

Die schnellen Tests werden mit PHPUnit umgesetzt. Integrationstestumgebung und Pipeline sind weiterhin geplant und noch nicht eingerichtet. Ziel ist, die fachlichen Regeln und möglichst viel des Betriebs lokal und in GitHub Actions reproduzierbar zu prüfen, ohne produktive Nextcloud-Daten, E-Mails oder Cyon-Zugänge zu verwenden.

### Testbare Struktur

- Die fachliche Logik für Zeiträume, Priorisierung, Kontaktzuordnung und Benachrichtigungen wird von CalDAV-/CardDAV-Zugriff, Uhr, Mailversand und Dateisystem getrennt. Diese Schnittstellen können in schnellen Tests durch kontrollierbare Testimplementierungen ersetzt werden.
- Die aktuelle Zeit wird innerhalb des Programms über eine austauschbare Uhr bezogen. Tests geben feste Zeitpunkte einschliesslich Zeitzone vor; im Betrieb wird ausschliesslich die Systemzeit verwendet. Ein Test-Zeitpunkt darf nicht über eine öffentlich erreichbare Anfrage gesetzt werden können.
- Tests verwenden ausschliesslich synthetische Kontakte, reservierte Beispieladressen und eigene Kalendereinträge. Testkonfiguration und Zugangsdaten sind von der Produktion getrennt; das Testziel darf keine produktiven Endpunkte verwenden.
- Das Auswerten der beim Abruf gültigen Webseite muss ebenfalls mit einer kontrollierbaren Uhr testbar sein. Das ist nötig, um auch einen ausgebliebenen oder fehlgeschlagenen Lauf zu prüfen, ohne bis zum nächsten Tag zu warten.

### Schnelle automatisierte Tests

Für die Umsetzung wird eine PHP-Testsuite, beispielsweise PHPUnit, mit versionierten Testdaten für Kalender und Kontakte vorgesehen. Sie läuft ohne Nextcloud, Mailserver und Cyon und prüft mindestens:

- **Dienstzeiten:** unmittelbar vor, genau um und nach 06:00 Uhr, mehrere aufeinanderfolgende Tage und mehrtägige Ganztageseinträge. Auch Monats- und Jahreswechsel sowie beide Zeitumstellungen in `Europe/Zurich` werden mit festen Daten geprüft.
- **Priorität und Zuordnung:** kürzerer Eintrag, späterer Erstellungszeitpunkt, aufsteigend sortierte UUID als letzte Entscheidung und unabhängiges Ergebnis bei anderer Eingabereihenfolge. Einladungsantworten verändern die Auswahl nicht; fehlende oder mehrdeutige Personen und Kontakte führen zu einem Hinweis statt einer falschen Nummer.
- **Erinnerungen und Lücken:** Standardwerte eine Woche und ein Monat sowie abweichende Konfigurationen, Grenzen der Prüfzeiträume und durchgehende Abdeckung. Empfänger, Absender, Inhalt und Zeitpunkt der erzeugten Nachrichten werden überprüft.
- **Webseite und Templates:** genau ein ausgewählter Kontakt mit Vorname und Telefonnummer, keine anderen Kontakt- oder Kalenderdaten, korrektes Escaping und anpassbare Templates. Ohne gültigen Kontakt wird eine entsprechende Meldung statt einer Nummer angezeigt. Vor und nach Ablauf sowie bei ausgebliebenem oder fehlgeschlagenem Lauf darf keine abgelaufene Nummer erscheinen.
- **Fehler und Konfiguration:** ungültige oder fehlende Angaben, nicht erreichbare Dienste, Versand- und Schreibfehler sowie fehlende Berechtigungen. Die Tests prüfen Fehlermeldung, Fehlerprotokoll und Exit-Status; auch ein gescheiterter E-Mail-Versand darf nicht als Erfolg gelten.

### Lokaler Integrationstest

Eine mit Docker Compose startbare, vom produktiven Betrieb getrennte Testumgebung soll die gleiche PHP-Version und die benötigten Erweiterungen wie die Zielumgebung verwenden. Sie umfasst:

1. Eine isolierte Nextcloud-Testinstanz mit einem Testaccount, einem Kalender und zwei Adressbüchern. Ein reproduzierbarer Setup-Schritt richtet sie ein und lädt synthetische CalDAV-/CardDAV-Testdaten, einschliesslich eines ganztägigen mehrtägigen Eintrags, einer Vertretung und einer Lücke.
2. Einen PHP-Container, der das Programm mit einer separaten Testkonfiguration und einem temporären Ausgabeverzeichnis ausführt. Eine lokal bereitgestellte Webseite wird über HTTP geprüft; das öffentliche Verzeichnis darf weder Konfiguration noch Zugangsdaten enthalten.
3. Einen lokalen Mailfänger. Da das Programm `mail()` verwendet, muss der Testcontainer dessen Sendmail-Schnittstelle ausdrücklich auf einen lokalen Transport zum Mailfänger umleiten; ein SMTP-Mailfänger allein fängt `mail()` nicht automatisch ab. Der Test überprüft die abgefangenen Nachrichten und ihre Empfänger. Eine Testkonfiguration, die E-Mails ins Internet senden könnte, ist nicht zulässig.

Der Test startet die Dienste, wartet auf deren tatsächliche Bereitschaft und führt den täglichen Programmlauf mit einer festen Testzeit aus. Er prüft die angezeigte Person, E-Mails für Erinnerung und Administration, die Gültigkeit der Webseite und den Exit-Status. Danach ändert er Kalender oder Kontakt und führt einen weiteren Lauf aus: Die Änderung darf erst dann sichtbar werden. Ein weiterer Fall simuliert einen fehlgeschlagenen Datenzugriff und prüft die Fehlermeldung sowie das Ablaufen der Anzeige. Die Testdaten und Container werden anschliessend wieder entfernt. Dienstversionen und Setup werden für reproduzierbare lokale und CI-Läufe festgelegt.

### Pipeline und Deployment

- Bei Pull Requests und Pushes führt GitHub Actions Composer-Validierung, PHP-Syntaxprüfung, schnelle Tests und den lokalen Integrationstest aus. Dieselben Testbefehle sollen auch lokal verfügbar und dokumentiert sein.
- Erst wenn die Prüfungen erfolgreich sind, darf der Deployment-Job bei einem Push auf `main` starten. Pull Requests, insbesondere aus Forks, erhalten keine Deployment-Secrets und können weder auf Cyon noch auf produktive Nextcloud-Daten zugreifen.
- Die Verarbeitung der Crontab und das Deployment werden unabhängig vom echten Server getestet: Eine isolierte Test-Crontab mit fremden Einträgen und ein temporäres Zielverzeichnis prüfen wiederholtes Ausführen, Erhalt fremder Einträge und Daten sowie Fehler beim Lesen oder Schreiben. Der Test darf weder die echte Crontab noch produktive Serverpfade verändern.
- Ein automatisierter Test kann nicht nachweisen, dass Cyon `mail()` zustellt, der Cron-Dienst die richtige Zeitzone verwendet oder der produktive Server erreichbar ist. Vor dem Pilotbetrieb werden deshalb auf Cyon mit ausdrücklich freigegebenen Testempfängern ein kontrollierter Mailversand, der Cron-Aufruf und die Anzeige geprüft; Zugangsdaten und personenbezogene Daten dürfen dabei nicht in Logs oder Testartefakten erscheinen.
