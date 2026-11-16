<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\CalendarParser;
use PHPUnit\Framework\TestCase;

final class CalendarParserTest extends TestCase
{
    public function testRecurrenceExceptionsAndAttendeeStatuses(): void
    {
        $ical = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n"
            . "BEGIN:VEVENT\r\nUID:recurring\r\nDTSTART;VALUE=DATE:20261101\r\nDTEND;VALUE=DATE:20261102\r\n"
            . "RRULE:FREQ=DAILY;COUNT=4\r\nEXDATE;VALUE=DATE:20261102\r\n"
            . "CREATED:20260101T000000Z\r\nATTENDEE;PARTSTAT=DECLINED:mailto:a@example.test\r\nEND:VEVENT\r\n"
            . "BEGIN:VEVENT\r\nUID:recurring\r\nRECURRENCE-ID;VALUE=DATE:20261103\r\n"
            . "DTSTART;VALUE=DATE:20261103\r\nDTEND;VALUE=DATE:20261104\r\n"
            . "ATTENDEE;PARTSTAT=ACCEPTED:mailto:b@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n";
        $events = (new CalendarParser())->parse($ical, new DateTimeImmutable('2026-11-01'), new DateTimeImmutable('2026-11-06'));
        self::assertSame(['2026-11-01', '2026-11-03', '2026-11-04'], array_map(static fn ($event) => $event->start, $events));
        self::assertSame(['a@example.test'], $events[0]->attendees);
        self::assertSame(['b@example.test'], $events[1]->attendees);
        self::assertSame('2026-11-04', $events[1]->end);
    }

    public function testInvalidEntriesAreReportedWithoutDiscardingValidOnes(): void
    {
        $ical = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n"
            . "BEGIN:VEVENT\r\nUID:bad\r\nDTSTART:20261101T080000Z\r\nEND:VEVENT\r\n"
            . "BEGIN:VEVENT\r\nUID:good\r\nDTSTART;VALUE=DATE:20261101\r\nATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n";
        $parser = new CalendarParser();
        $events = $parser->parse($ical, new DateTimeImmutable('2026-11-01'), new DateTimeImmutable('2026-11-03'));
        self::assertCount(1, $events);
        self::assertSame('good', $events[0]->uid);
        self::assertNotEmpty($parser->issues());
    }

    public function testCancelledOccurrenceIsNotAssigned(): void
    {
        $ical = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n"
            . "BEGIN:VEVENT\r\nUID:recurring\r\nDTSTART;VALUE=DATE:20261101\r\nRRULE:FREQ=DAILY;COUNT=2\r\nATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\n"
            . "BEGIN:VEVENT\r\nUID:recurring\r\nRECURRENCE-ID;VALUE=DATE:20261102\r\nDTSTART;VALUE=DATE:20261102\r\nSTATUS:CANCELLED\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n";
        $events = (new CalendarParser())->parse($ical, new DateTimeImmutable('2026-11-01'), new DateTimeImmutable('2026-11-04'));
        self::assertCount(1, $events);
        self::assertSame('2026-11-01', $events[0]->start);
    }

    public function testAllDayDurationCoversAllDays(): void
    {
        $ical = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\n"
            . "BEGIN:VEVENT\r\nUID:long\r\nDTSTART;VALUE=DATE:20261101\r\nDURATION:P3D\r\nATTENDEE:mailto:a@example.test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n";
        $events = (new CalendarParser())->parse($ical, new DateTimeImmutable('2026-11-01'), new DateTimeImmutable('2026-11-05'));
        self::assertSame('2026-11-04', $events[0]->end);
    }
}
