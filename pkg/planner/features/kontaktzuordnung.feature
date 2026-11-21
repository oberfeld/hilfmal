# language: de
Funktionalität: Kontaktzuordnung zu einem Dienst
  Pro Dienst zählt zuerst eine gültige eingeladene Person, dann der Titel,
  zuletzt die organisierende Person. Eine mehrdeutige Zuordnung wird gemeldet.

  Grundlage:
    Angenommen das Adressbuch enthält
      | Vorname | Nachname      | Telefon | E-Mails                                                    |
      | Ada     | Müller        | 123     | ungültig, ada@example.test, ada.arbeit@example.test        |
      | Ada     | Meier         | 456     | meier@example.test                                         |
      | Berta   | Zeller        | 789     | berta@example.test                                         |
      | Anna    | Huber         | 111     | anna@example.test                                          |
      | Nick    | Voll          | 222     | nick@example.test                                          |
      | Nick    | Unvollständig |         | unvollstaendig@example.test                                |

  Szenariogrundriss: <Fall>
    Angenommen ein Dienst mit Titel "<Titel>", Organisator "<Organisator>" und Teilnehmenden "<Teilnehmende>"
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "<Erwartete E-Mail>"
    Und es gibt keinen Kontakt-Hinweis

    Beispiele:
      | Fall                                                       | Titel                  | Organisator            | Teilnehmende                           | Erwartete E-Mail        |
      | Ein eindeutiger Vorname im Titel genügt                     | Berta 2 ist zuständig  |                        |                                        | berta@example.test      |
      | Der Nachname unterscheidet Personen mit gleichem Vornamen  | ADA MÜLLER ist da      |                        |                                        | ada@example.test        |
      | Ein Nachname unterscheidet auch verschiedene Vornamen       | Ada Meier und Berta    |                        |                                        | meier@example.test      |
      | Unvollständige Kontakte machen einen Vornamen nicht mehrdeutig | Nick ist zuständig  |                        |                                        | nick@example.test       |
      | Die eingeladene Person hat Vorrang vor dem Titel           | Berta ist zuständig    |                        | ada.arbeit@example.test                | ada.arbeit@example.test |
      | Der Titel hat Vorrang vor dem Organisator                  | Berta ist zuständig    | meier@example.test     | MEIER@example.test                     | berta@example.test      |
      | Eine zweite Adresse macht den Organisator nicht zum Gast    | Berta ist zuständig    | ada@example.test       | ada.arbeit@example.test                | berta@example.test      |
      | Ohne andere Person wird der Organisator gewählt             | Pikettdienst           | berta@example.test     |                                        | berta@example.test      |
      | Ein unbekannter Gast verhindert den Organisator nicht       | Pikettdienst           | berta@example.test     | unbekannt@example.test                 | berta@example.test      |
      | Ein unvollständiger Gast verhindert den Organisator nicht   | Pikettdienst           | berta@example.test     | unvollstaendig@example.test            | berta@example.test      |
      | Der Organisator kann selbst als Gast aufgeführt sein        | Pikettdienst           | berta@example.test     | BERTA@example.test                     | berta@example.test      |
      | Ein anderer gültiger Gast geht dem Organisator vor          | Pikettdienst           | berta@example.test     | berta@example.test, meier@example.test | meier@example.test      |
      | Eine gültige Einladung funktioniert ohne bekannten Organisator | Pikettdienst       | unbekannt@example.test | meier@example.test                     | meier@example.test      |

  Szenariogrundriss: <Fall>
    Angenommen ein Dienst mit Titel "<Titel>", Organisator "<Organisator>" und Teilnehmenden "<Teilnehmende>"
    Wenn die zuständige Person ermittelt wird
    Dann wird keine Person zugeordnet
    Und es gibt einen Kontakt-Hinweis

    Beispiele:
      | Fall                                                | Titel                    | Organisator            | Teilnehmende                |
      | Zwei vollständige Namen bleiben mehrdeutig         | Ada Müller und Ada Meier |                        |                             |
      | Verschiedene Vornamen ohne Nachnamen bleiben mehrdeutig | Ada und Berta        |                        |                             |
      | Ein Teil eines längeren Wortes ist kein Name       | Johanna ist zuständig    |                        |                             |
      | Ohne vollständigen Kontakt gibt es einen Hinweis   | Pikettdienst             | unbekannt@example.test | unvollstaendig@example.test |

  Szenario: Gleiche Vornamen ohne Nachname führen zur Auswahl nach E-Mail
    Angenommen ein Dienst mit Titel "Ada ist zuständig", Organisator "berta@example.test" und Teilnehmenden ""
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "ada@example.test"
    Und es gibt einen Kontakt-Hinweis
    Und der Kontakt-Hinweis enthält "ada@example.test"

  Szenario: Derselbe Name als Vor- und Nachname zählt nicht zweimal
    Angenommen das Adressbuch enthält
      | Vorname | Nachname | Telefon | E-Mails             |
      | Alex    | Alex     | 123     | z@example.test      |
      | Alex    | Test     | 456     | andere@example.test |
    Und ein Dienst mit Titel "Alex ist zuständig", Organisator "" und Teilnehmenden ""
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "andere@example.test"
    Und es gibt einen Kontakt-Hinweis
    Und der Kontakt-Hinweis enthält "andere@example.test"

  Szenario: Derselbe Name kann Vor- und Nachname eindeutig bezeichnen
    Angenommen das Adressbuch enthält
      | Vorname | Nachname | Telefon | E-Mails             |
      | Alex    | Alex     | 123     | alex@example.test   |
      | Alex    | Test     | 456     | andere@example.test |
    Und ein Dienst mit Titel "Alex Alex ist zuständig", Organisator "" und Teilnehmenden ""
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "alex@example.test"
    Und es gibt keinen Kontakt-Hinweis

  Szenario: Bei gleichem Vor- und Nachnamen wird eine Person ausgewählt und gemeldet
    Angenommen das Adressbuch enthält
      | Vorname | Nachname | Telefon | E-Mails             |
      | Alex    | Test     | 123     | alex@example.test   |
      | Alex    | Test     | 456     | andere@example.test |
    Und ein Dienst mit Titel "Alex Test ist zuständig", Organisator "" und Teilnehmenden ""
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "alex@example.test"
    Und es gibt einen Kontakt-Hinweis
    Und der Kontakt-Hinweis enthält "alex@example.test"

  Szenario: Die Priorität zwischen Diensten bleibt erhalten
    Angenommen ein kurzer Dienst mit Organisator "berta@example.test" und ein längerer Dienst mit teilnehmender Person "meier@example.test"
    Wenn die zuständige Person ermittelt wird
    Dann ist die E-Mail der zuständigen Person "berta@example.test"
    Und es gibt keinen Kontakt-Hinweis
