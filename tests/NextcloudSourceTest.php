<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\DavTransport;
use Hilfmal\NextcloudSource;
use PHPUnit\Framework\TestCase;

final class NextcloudSourceTest extends TestCase
{
    public function testReadsDavDataAndUsesGivenNameAndEmail(): void
    {
        $transport = new FakeDav();
        $source = new NextcloudSource($transport, 'calendar', 'contacts', 'admins');
        self::assertSame(['admin@example.test'], $source->admins());
        $contacts = $source->contacts();
        self::assertCount(1, $contacts);
        self::assertSame('Ada', $contacts[0]->name);
        self::assertSame('+41 12', $contacts[0]->phone);
        self::assertSame('ada@example.test', $contacts[0]->email);
        $events = $source->events(new DateTimeImmutable('2026-11-01'), new DateTimeImmutable('2026-11-10'));
        self::assertCount(1, $events);
        self::assertSame('ada@example.test', $events[0]->attendees[0]);
    }

    public function testDavPropertyErrorsAreNotIgnored(): void
    {
        $transport = new FakeDav();
        $transport->broken = true;
        $source = new NextcloudSource($transport, 'calendar', 'contacts', 'admins');
        $this->expectException(\RuntimeException::class);
        $source->contacts();
    }
}

final class FakeDav implements DavTransport
{
    public bool $broken = false;
    public function report(string $url, string $body): string
    {
        if ($this->broken) {
            return '<d:multistatus xmlns:d="DAV:"><d:response><d:propstat><d:prop/><d:status>HTTP/1.1 403 Forbidden</d:status></d:propstat></d:response></d:multistatus>';
        }
        $data = match ($url) {
            'admins' => "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Admin\r\nEMAIL:admin@example.test\r\nEND:VCARD\r\n",
            'contacts' => "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Lovelace;Ada;;;\r\nFN:Ada Lovelace\r\nTEL:+41 12\r\nEMAIL:ada@example.test\r\nEND:VCARD\r\n",
            'calendar' => "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\nBEGIN:VEVENT\r\nUID:test\r\nDTSTART;VALUE=DATE:20261101\r\nDTEND;VALUE=DATE:20261102\r\nATTENDEE:mailto:ada@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
        };
        $tag = $url === 'calendar' ? 'c:calendar-data' : 'a:address-data';
        return '<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:response><d:propstat><d:prop><' . $tag . '>' . htmlspecialchars($data, ENT_XML1) . '</' . $tag . '></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>';
    }
}
