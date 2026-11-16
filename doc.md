# Hilfmal: Anleitung für Mitwirkende und Administration

Diese Anleitung beschreibt den vorgesehenen Ablauf. Das Programm und die automatischen Benachrichtigungen befinden sich noch in der Konzeptphase. Die Idee hinter Hilfmal steht im [Konzept](konzept.md), die technischen Anforderungen stehen in der [Architektur](architektur.md).

## Für Mitwirkende

### Dienst und Erreichbarkeit

Hilfmal unterstützt Jugendliche und Anwohnende bei Situationen in den Gemeinschaftsräumen. Die Mitwirkung ist freiwillig. Die Koordination verwaltet den Dienstplan und die Kontaktdaten.

Ein Dienst dauert von 06:00 Uhr bis 06:00 Uhr am nächsten Tag. Die Webseite zeigt den Vornamen und die Telefonnummer der eingeteilten Person; sie wird einmal täglich um 06:00 Uhr aktualisiert. Änderungen am Dienstplan erscheinen erst nach dem nächsten erfolgreichen Lauf. Eine Erinnerung per E-Mail weist auf den bevorstehenden Dienst hin.

Bei Verhinderung organisiert die eingeteilte Person eine Vertretung, zum Beispiel über eine WhatsApp- oder Nextcloud-Talk-Gruppe, und informiert die Administration, damit diese den Dienstplan anpasst. Da die Webseite nur einmal täglich aktualisiert wird, müssen kurzfristige Änderungen auch ausserhalb der Webseite kommuniziert werden.

### Vorgehen bei Meldungen

Höre dir die Situation an und entscheide, ob ein Einsatz vor Ort nötig ist. Erinnere die Jugendlichen wenn möglich zunächst an ihre Eigenverantwortung und die Hausordnung. Je nach Situation kannst du Anpassungen vorschlagen oder verlangen, etwa die Musik leiser zu stellen oder Gäste wegzuschicken. Wenn nötig, kann ein Treffen beendet werden.

Der Pikettdienst ist eine Anlaufstelle, keine Aufsicht oder Hilfspolizei. Handle ruhig und verhältnismässig und vermeide Eskalationen. Bei akuter Gefahr oder wenn du die Situation nicht allein bewältigen kannst, ziehe die Polizei bei. Körperliche Interventionen und die eigene Gefährdung gehören nicht zum Auftrag.

Halte Einsätze knapp fest: Datum, Uhrzeit, kurze Beschreibung und getroffene Massnahme.

### Kalender und Austritt

Die Administration kann den Nextcloud-Kalender für Mitwirkende freigeben. Um ihn auf dem Handy anzuzeigen, richte den freigegebenen Nextcloud-Kalender in der Kalender-App deines Geräts ein; das genaue Vorgehen hängt vom Gerät und der verwendeten App ab.

Wenn du nicht mehr mitwirken möchtest, informiere die Koordination. Sie sorgt dafür, dass deine künftigen Dienste neu besetzt und deine Kontaktdaten aus der Liste der Mitwirkenden entfernt werden.

## Für die Administration

### Hilfmal-Account und Freigaben

Auf der Chischte gibt es den Account «Hilfmal». Er besitzt den Dienstplan-Kalender und die Adressbücher für Mitwirkende und Administration. Die Administration pflegt sie möglichst über persönliche Accounts und entsprechende Freigaben.

Das Passwort des Hilfmal-Accounts wird unter den Admin-Personen geteilt und in einem Passwortprogramm gespeichert. Wenn eine Person mit Zugang ausscheidet, wird das Passwort idealerweise gewechselt und das neue Passwort den verbleibenden Admin-Personen mitgeteilt.

Für den Zugriff des Programms wird zusätzlich ein eigenes App-Passwort erzeugt. Wie dieses geschützt hinterlegt und beim Deployment bereitgestellt wird, ist in der [Architektur](architektur.md) beschrieben.

### Mitwirkende verwalten

1. Pflege im Adressbuch «Mitwirkende» für jede Person einen Kontakt mit Vorname, Name, eindeutiger E-Mail-Adresse und Telefonnummer.
2. Kopiere bei Bedarf einen bestehenden Kontakt aus dem Adressbuch «Konten» in «Mitwirkende» und ergänze fehlende Angaben.
3. Teile das Adressbuch mit den zuständigen Admin-Personen mit Schreibrecht. Eine Freigabe mit Leserecht für alle Mitwirkenden ist optional.
4. Halte die Angaben aktuell und entferne den Kontakt beim Austritt der Person.

### Administrationsadressen verwalten

Pflege im separaten Adressbuch für die Administration die Kontakte mit den E-Mail-Adressen, an die Hinweise zu Lücken und unvollständigen Dienstplaneinträgen gesendet werden. Teile das Adressbuch mit den zuständigen Admin-Personen mit Schreibrecht und halte die Adressen aktuell.

### Dienstplan pflegen

1. Erfasse jeden Dienst im Kalender des Accounts «Hilfmal» als ganztägigen Eintrag und füge genau eine Person aus dem Adressbuch «Mitwirkende» als eingeladene Person hinzu. Ihre E-Mail-Adresse muss mit derjenigen im Adressbuch übereinstimmen. Ob sie die Einladung annimmt oder ablehnt, wird nicht berücksichtigt.
2. Stelle sicher, dass jeder Tag abgedeckt ist. Jeder eingetragene Tag gilt von 06:00 Uhr bis 06:00 Uhr des Folgetags. Mehrtägige ganztägige Einträge können mehrere Tage abdecken.
3. Für eine Vertretung kann ein überlappender Eintrag erstellt werden. Der kürzere Eintrag hat Vorrang; bei gleicher Dauer der später erstellte. Prüfe, dass die Vertretung die nötigen Kontaktdaten hat.
4. Beachte, dass die Webseite nur einmal täglich um 06:00 Uhr aktualisiert wird. Kurzfristige Änderungen müssen zusätzlich ausserhalb der Webseite kommuniziert werden.
5. Prüfe die E-Mail-Hinweise zu Lücken und unvollständigen Einträgen und korrigiere den Dienstplan oder die Kontaktdaten.

### Austritt einer Admin-Person

- Entferne ihr Schreibrecht auf die Adressbücher «Mitwirkende» und Administration sowie auf den Dienstplan-Kalender.
- Entferne ihren Kontakt aus dem Administrationsadressbuch.
- Wechsle das Hilfmal-Account-Passwort, falls sie es kannte, und teile das neue Passwort den verbleibenden Admin-Personen mit.
