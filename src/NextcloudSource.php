<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;
use DOMDocument;
use DOMXPath;
use RuntimeException;
use Sabre\VObject\Component\VCard;
use Sabre\VObject\Reader;

final class NextcloudSource implements DataSource
{
    /** @var list<string> */
    private array $issues = [];

    public function __construct(private readonly DavTransport $transport, private readonly string $calendarUrl, private readonly string $contactsUrl, private readonly string $adminsUrl)
    {
    }

    /** @return list<string> */
    private function query(string $url, bool $calendar): array
    {
        $body = $calendar
            ? '<?xml version="1.0"?><c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav"><d:prop><c:calendar-data/></d:prop><c:filter><c:comp-filter name="VCALENDAR"><c:comp-filter name="VEVENT"/></c:comp-filter></c:filter></c:calendar-query>'
            : '<?xml version="1.0"?><a:addressbook-query xmlns:d="DAV:" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:prop><a:address-data/></d:prop></a:addressbook-query>';
        $xml = $this->transport->report($url, $body);
        $doc = new DOMDocument();
        $previous = libxml_use_internal_errors(true);
        try {
            if (!$doc->loadXML($xml, LIBXML_NONET)) {
                throw new RuntimeException('Ungültige DAV-Antwort.');
            }
        } finally {
            libxml_clear_errors();
            libxml_use_internal_errors($previous);
        }
        $xpath = new DOMXPath($doc);
        $xpath->registerNamespace('d', 'DAV:');
        $xpath->registerNamespace('c', 'urn:ietf:params:xml:ns:caldav');
        $xpath->registerNamespace('a', 'urn:ietf:params:xml:ns:carddav');
        if ($doc->documentElement?->localName !== 'multistatus' || $doc->documentElement?->namespaceURI !== 'DAV:') {
            throw new RuntimeException('Ungültige DAV-Antwort.');
        }
        $result = [];
        foreach ($xpath->query('/d:multistatus/d:response') as $response) {
            $statuses = $xpath->query('./d:status | ./d:propstat/d:status', $response);
            if ($statuses->length === 0) {
                throw new RuntimeException('DAV-Antwort ohne Status.');
            }
            foreach ($statuses as $status) {
                if (!preg_match('/^HTTP\/\S+ 2\d\d(?:\s|$)/', trim($status->textContent))) {
                    throw new RuntimeException('DAV-Ressource konnte nicht gelesen werden.');
                }
            }
            $data = $xpath->query($calendar ? './d:propstat/d:prop/c:calendar-data' : './d:propstat/d:prop/a:address-data', $response);
            if ($data->length !== 1) {
                throw new RuntimeException('DAV-Antwort enthält keine eindeutigen Daten.');
            }
            $result[] = $data->item(0)->textContent;
        }
        return $result;
    }

    public function admins(): array
    {
        $emails = [];
        foreach ($this->query($this->adminsUrl, false) as $data) {
            $card = Reader::read($data);
            if (!$card instanceof VCard) {
                throw new RuntimeException('Ungültiger Administrationskontakt.');
            }
            foreach ($card->select('EMAIL') as $email) {
                $emails[] = trim((string) $email);
            }
        }
        return $emails;
    }

    public function contacts(): array
    {
        $contacts = [];
        foreach ($this->query($this->contactsUrl, false) as $data) {
            $card = Reader::read($data);
            if (!$card instanceof VCard) {
                throw new RuntimeException('Ungültiger Kontakt.');
            }
            $name = isset($card->N) ? ($card->N->getParts()[1] ?? '') : '';
            $phone = isset($card->TEL) ? (string) $card->TEL : '';
            foreach ($card->select('EMAIL') as $email) {
                $contacts[] = new Contact($name, $phone, trim((string) $email));
            }
        }
        return $contacts;
    }

    public function events(DateTimeImmutable $from, DateTimeImmutable $to): array
    {
        $events = [];
        $this->issues = [];
        $parser = new CalendarParser();
        foreach ($this->query($this->calendarUrl, true) as $data) {
            array_push($events, ...$parser->parse($data, $from, $to));
            array_push($this->issues, ...$parser->issues());
        }
        return $events;
    }

    public function issues(): array
    {
        return $this->issues;
    }
}
