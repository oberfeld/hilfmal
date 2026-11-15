# Hilfmal - der Pikettdienst für Jugendunruhen in Gemeinschafträumen.

**Hilfmal** ist der freiwillige Hilfsdienst für die Gemeinschaftsräume der WBGO. Mithelfen können alle, die sich dafür engagieren möchten. Eltern dürften dabei eine besonders angesprochen sein, weil sie eine Aufsichtspflicht haben.

## Ausgangslage und Ziel

Die Gemeinschaftsräume der WBGO werden regelmässig von Jugendlichen genutzt. Das ist erwünscht, kann aber auch zu Lärm, Konflikten oder Situationen führen, die Jugendliche nicht mehr allein bewältigen können oder die Anwohnenden nicht allein klären möchten. Freiwillige Helferinnen und Helfer unterstützen die Jugendlichen, wirken auf die Einhaltung der Regeln hin und helfen bei Bedarf, ein Treffen geordnet zu beenden.

Der Pikettdienst ist eine Anlaufstelle, keine Aufsicht oder Hilfspolizei. Er soll ruhig und verhältnismässig handeln und Eskalationen vermeiden.

## Organisation und Kontakt

- Interessierte Freiwillige übernehmen den Pikettdienst nach einem Dienstplan.
- Eine Koordinationsperson verwaltet die Liste der Mitwirkenden und organisiert die Planung.
- Die eingeteilte Person organisiert bei Verhinderung eine Vertretung, zum Beispiel über eine WhatsApp- oder Nextcloud-Talk-Gruppe.
- Jugendliche und Anwohnende können die diensthabende Person telefonisch kontaktieren.
- Die jeweils aktuellen Kontaktdaten werden online veröffentlicht. Ein QR-Code im Gemeinschaftsraum führt zur entsprechenden Webseite.
- Öffentlich angezeigt werden ausschliesslich Vorname und Telefonnummer der diensthabenden Person.

## Vorgehen bei Meldungen

Die diensthabende Person hört sich die Situation an und entscheidet, ob ein Einsatz vor Ort nötig ist. Wenn möglich, werden die Jugendlichen zunächst an ihre Eigenverantwortung und die Hausordnung erinnert. Je nach Situation kann die helfende Person Anpassungen vorschlagen oder verlangen, etwa die Musik leiser zu stellen oder Gäste wegzuschicken. Wenn nötig, kann ein Treffen beendet werden.

Bei akuter Gefahr oder wenn die Situation nicht alleine bewältigt werden kann, wird die Polizei beigezogen. Körperliche Interventionen und die eigene Gefährdung gehören nicht zum Auftrag des Pikettdienstes.

Einsätze werden knapp festgehalten: Datum, Uhrzeit, kurze Beschreibung und getroffene Massnahme.

## Technische Lösung für den Pilotbetrieb

Die Lösung besteht aus einem Nextcloud-Kalender, einer Liste der Mitwirkenden in Nextcloud, einer Webseite zur Anzeige des aktuellen Pikettdienstes und automatischen Erinnerungen.

- Der Dienstplan wird in einem geteilten Nextcloud-Kalender geführt.
- Die Webseite zeigt automatisch den Vornamen und die Telefonnummer der aktuell eingeteilten Person.
- Ein QR-Code an den Eingängen führt zur Webseite.
- Ein automatisierter Erinnerungsdienst erinnert die eingeteilte Person an den bevorstehenden Dienst und daran, bei Verhinderung eine Vertretung zu organisieren.
- Ein weiterer Hinweis informiert die Koordination über nicht besetzte Dienste.
- Eine Liste in Nextcloud führt die Mitwirkenden und ihre Telefonnummern.

## Lösungsentscheide

### Liste der Mitwirkenden

Bevorzugt wird eine Kontaktgruppe in Nextcloud Contacts. Die Kontakte enthalten Vorname, Telefonnummer und eine für die Zuordnung eindeutige E-Mail-Adresse. Die diensthabende Person wird einzeln als Teilnehmerin oder Teilnehmer zum jeweiligen Kalendereintrag eingeladen. Ein Script liest den Dienstplan über CalDAV und die Kontakte über CardDAV und ordnet die Person anhand der E-Mail-Adresse zu.

Ob Einladungen und Antworten in der verwendeten Nextcloud wie gewünscht funktionieren, wird vor dem Pilotbetrieb getestet. Dabei ist auch festzulegen, wie mit ausstehenden oder abgelehnten Einladungen umgegangen wird. Bei fehlender oder mehrdeutiger Zuordnung darf keine falsche Telefonnummer veröffentlicht werden.

Als Alternativen wurden Nextcloud Tables mit einer Personenkennung im Kalendereintrag sowie eine Tabellen- oder JSON-Datei verworfen. Sie ermöglichen ebenfalls das Auslesen per Script, erfordern aber eine separat gepflegte Zuordnung zum Kalendereintrag und nutzen die Kalender-Einladungsfunktion nicht direkt.

### Service Account
Ein eigener Nextcloud-Account ohne Adminrechte besitzt Kalender und Adressbuch.
Die Koordination pflegt beides über ihren persönlichen Account und Freigaben.
Das Script nutzt den Service Account mit einem eigenen App-Passwort.
Es hat damit die Rechte des Eigentümers; das App-Passwort ist sicher aufzubewahren.

## Anleitung für Mitwirkende

Eine Seite auf den «Grünen Seiten» erklärt den Ablauf für alle Mitwirkenden. Sie enthält:

- Motivation für den Einsatz
- Empfehlungen zum Vorgehen
- Vorgehen bei der Organisation von Vertretungen
- Hinweise zur Protokollierung eines Einsatzes
- Anleitung zum Eintragen des Kalenders auf dem Handy
- Informationen zum Austritt aus der Gruppe

## Pilot und Auswertung

Nach einem Pilotjahr prüft die WBGO:

- Wie oft wurde der Pikettdienst kontaktiert und wie oft war ein Einsatz nötig?
- Wie oft mussten externe Stellen beigezogen werden?
- Fühlen sich Jugendliche unterstützt und hat sich die Situation für Anwohnende verbessert?
- Reicht die Zahl der Mitwirkenden aus?
- Fühlen sich die Mitwirkenden den Aufgaben gewachsen?
- Ist der Koordinationsaufwand angemessen?
- Wird künftig mehr oder weniger technische Unterstützung benötigt?
