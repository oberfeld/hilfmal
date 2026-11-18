# language: de
Funktionalität: Täglicher Programmlauf
  Die Anwendung verarbeitet Kalender und Adressbücher, versendet nötige Hinweise
  und veröffentlicht die aktuelle diensthabende Person.

  Szenario: Erinnerung und Lückenhinweis werden versendet und der Dienstplan veröffentlicht
    Angenommen die Anwendung läuft am "2026-11-01T06:00:00+01:00" mit 7 Erinnerungstagen und Intervall "P1M"
    Und Administratoren sind "admin@example.test"
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
      | Person  | 456     | person@example.test  |
    Und der Kalender enthält die Dienste
      | UID      | Von        | Bis        | E-Mail               |
      | heute    | 2026-11-01 | 2026-11-02 | current@example.test |
      | erinnerung | 2026-11-08 | 2026-11-09 | person@example.test  |
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 0
    Und die Webseite zeigt "Aktuell"
    Und es wurde eine E-mail an "person@example.test" versendet welche im Body "2026-11-08" enthält
    Und es wurde eine E-mail an "admin@example.test" versendet welche im Body "2026-11-02" enthält
    Und der Kalender wurde von "2026-10-31T00:00:00+01:00" bis "2026-12-01T00:00:00+01:00" gelesen

  Szenario: Ein vollständiger Dienstplan erzeugt keine E-Mails
    Angenommen die Anwendung läuft am "2026-11-01T06:00:00+01:00" mit 7 Erinnerungstagen und Intervall "P1M"
    Und Administratoren sind "admin@example.test"
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
    Und der Kalender enthält die Dienste
      | UID       | Von        | Bis        | E-Mail               |
      | abdeckung | 2026-11-01 | 2026-12-02 | current@example.test |
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 0
    Und die Webseite zeigt "Aktuell"
    Und es wurden keine E-Mails versendet

  Szenario: Bei einem Fehler beim Lesen der Administratoren wird trotzdem veröffentlicht
    Angenommen die Anwendung läuft am "2026-11-01T06:00:00+01:00" mit 7 Erinnerungstagen und Intervall "P1M"
    Und Administratoren können nicht gelesen werden
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
    Und der Kalender enthält die Dienste
      | UID       | Von        | Bis        | E-Mail               |
      | abdeckung | 2026-11-01 | 2026-12-02 | current@example.test |
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 1
    Und die Webseite zeigt "Aktuell"
    Und eine Fehlerbenachrichtigung wurde an "emergency@example.test" versucht

  Szenario: Die Webseite wird auch bei fehlgeschlagenem Mailversand veröffentlicht
    Angenommen die Anwendung läuft am "2026-11-01T06:00:00+01:00" mit 7 Erinnerungstagen und Intervall "P1M"
    Und Administratoren sind "admin@example.test"
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
      | Person  | 456     | person@example.test  |
    Und der Kalender enthält die Dienste
      | UID      | Von        | Bis        | E-Mail               |
      | heute    | 2026-11-01 | 2026-11-02 | current@example.test |
      | erinnerung | 2026-11-08 | 2026-11-09 | person@example.test  |
    Und der Mailversand schlägt fehl
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 1
    Und die E-Mail an "person@example.test" konnte nicht versendet werden
    Und die Webseite zeigt "Aktuell"

  Szenario: Ein Veröffentlichungsfehler wird an Administration und Notfalladresse gemeldet
    Angenommen die Anwendung läuft am "2026-11-01T06:00:00+01:00" mit 7 Erinnerungstagen und Intervall "P1M"
    Und Administratoren sind "admin@example.test"
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
      | Person  | 456     | person@example.test  |
    Und der Kalender enthält die Dienste
      | UID      | Von        | Bis        | E-Mail               |
      | heute    | 2026-11-01 | 2026-11-02 | current@example.test |
      | erinnerung | 2026-11-08 | 2026-11-09 | person@example.test  |
    Und die Veröffentlichung schlägt fehl
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 1
    Und eine Fehlerbenachrichtigung wurde an "admin@example.test" versucht
    Und eine Fehlerbenachrichtigung wurde an "emergency@example.test" versucht

  Szenario: Der gelesene Kalenderbereich verwendet den Schweizer lokalen Tagesbeginn
    Angenommen die Anwendung läuft am "2026-10-31T06:00:00+01:00" mit 0 Erinnerungstagen und Intervall "P1M"
    Und Administratoren sind "admin@example.test"
    Und Mitwirkende sind
      | Name    | Telefon | E-Mail               |
      | Aktuell | 123     | current@example.test |
    Und der Kalender enthält die Dienste
      | UID       | Von        | Bis        | E-Mail               |
      | abdeckung | 2026-10-30 | 2026-12-02 | current@example.test |
    Wenn die Anwendung ausgeführt wird
    Dann endet der Lauf mit Status 0
    Und der Kalender wurde von "2026-10-30T00:00:00+01:00" bis "2026-12-01T00:00:00+01:00" gelesen
