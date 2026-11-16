<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;
use DateTimeZone;
use RuntimeException;
use Sabre\VObject\Component\VCalendar;
use Sabre\VObject\DateTimeParser;
use Sabre\VObject\Reader;

final class CalendarParser
{
    /** @var list<string> */
    private array $issues = [];

    /** @return list<string> */
    public function issues(): array
    {
        return $this->issues;
    }

    /** @return list<Event> */
    public function parse(string $ical, DateTimeImmutable $from, DateTimeImmutable $to): array
    {
        $this->issues = [];
        $calendar = Reader::read($ical);
        if (!$calendar instanceof VCalendar) {
            throw new RuntimeException('Kalenderdaten sind kein VCALENDAR.');
        }
        foreach ($calendar->select('VEVENT') as $event) {
            if (!isset($event->UID, $event->DTSTART) || $event->DTSTART->hasTime() || (isset($event->DTEND) && $event->DTEND->hasTime())) {
                $this->issues[] = 'Kalendereintrag benötigt eine UID und ganztägige Datumsangaben.';
                $calendar->remove($event);
            }
        }
        $expanded = $calendar->expand($from, $to, new DateTimeZone('Europe/Zurich'));
        $result = [];
        foreach ($expanded->select('VEVENT') as $event) {
            if (isset($event->STATUS) && strtoupper((string) $event->STATUS) === 'CANCELLED') {
                continue;
            }
            $start = $event->DTSTART->getDateTime()->format('Y-m-d');
            if (isset($event->DTEND)) {
                $end = $event->DTEND->getDateTime()->format('Y-m-d');
            } elseif (isset($event->DURATION)) {
                $duration = (string) $event->DURATION;
                if (!preg_match('/^P(?:[0-9]+W|[0-9]+D)$/', $duration)) {
                    $this->issues[] = 'Ganztägiger Kalendereintrag hat eine ungültige Dauer.';
                    continue;
                }
                $end = (new DateTimeImmutable($start))->add(DateTimeParser::parseDuration($duration))->format('Y-m-d');
            } else {
                $end = (new DateTimeImmutable($start))->modify('+1 day')->format('Y-m-d');
            }
            if ($end <= $start) {
                $this->issues[] = 'Kalendereintrag hat einen ungültigen Zeitraum.';
                continue;
            }
            $attendees = [];
            foreach ($event->select('ATTENDEE') as $attendee) {
                $attendees[] = preg_replace('/^mailto:/i', '', (string) $attendee);
            }
            $result[] = new Event((string) $event->UID, $start, $end, $attendees, isset($event->CREATED) ? $event->CREATED->getDateTime()->format(DATE_ATOM) : null);
        }
        usort($result, static fn (Event $a, Event $b): int => [$a->start, $a->uid] <=> [$b->start, $b->uid]);
        return $result;
    }
}
