<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;
use DateTimeZone;

final class Planner
{
    public function serviceDate(DateTimeImmutable $now): string
    {
        $local = $now->setTimezone(new DateTimeZone('Europe/Zurich'));
        if ((int) $local->format('G') < 6) {
            $local = $local->modify('-1 day');
        }
        return $local->format('Y-m-d');
    }

    /** @param list<Event> $events @param list<Contact> $contacts */
    public function select(string $date, array $events, array $contacts): ?Contact
    {
        $candidates = array_values(array_filter($events, static fn (Event $event): bool => $event->start <= $date && $date < $event->end));
        $missingCreated = [];
        foreach ($candidates as $candidate) {
            $days = (new DateTimeImmutable($candidate->start))->diff(new DateTimeImmutable($candidate->end))->days;
            if ($candidate->created === null) {
                $missingCreated[$days] = true;
            }
        }
        usort($candidates, static function (Event $a, Event $b) use ($missingCreated): int {
            $aDays = (new DateTimeImmutable($a->start))->diff(new DateTimeImmutable($a->end))->days;
            $bDays = (new DateTimeImmutable($b->start))->diff(new DateTimeImmutable($b->end))->days;
            $duration = $aDays <=> $bDays;
            if ($duration !== 0) {
                return $duration;
            }
            if (!isset($missingCreated[$aDays]) && $a->created !== null && $b->created !== null) {
                $created = (new DateTimeImmutable($b->created)) <=> (new DateTimeImmutable($a->created));
                if ($created !== 0) {
                    return $created;
                }
            }
            return strcmp($a->uid, $b->uid);
        });
        foreach ($candidates as $event) {
            $contact = $this->resolve($event, $contacts);
            if ($contact !== null) {
                return $contact;
            }
        }
        return null;
    }

    /** @param list<Contact> $contacts */
    private function resolve(Event $event, array $contacts): ?Contact
    {
        if (count($event->attendees) !== 1) {
            return null;
        }
        $matches = array_values(array_filter($contacts, static fn (Contact $contact): bool => strcasecmp($contact->email, $event->attendees[0]) === 0));
        if (count($matches) !== 1 || trim($matches[0]->name) === '' || trim($matches[0]->phone) === '' || !filter_var($matches[0]->email, FILTER_VALIDATE_EMAIL)) {
            return null;
        }
        return $matches[0];
    }

    /** @param list<Event> $events @param list<Contact> $contacts @return list<string> */
    public function issues(array $events, array $contacts): array
    {
        $issues = [];
        foreach ($events as $event) {
            if ($this->resolve($event, $contacts) === null) {
                $issues[] = "Dienst {$event->uid} ({$event->start}): Kontakt fehlt oder ist nicht eindeutig.";
            }
        }
        return array_values(array_unique($issues));
    }

    /** @param list<Event> $events @param list<Contact> $contacts @return list<string> */
    public function gaps(string $start, string $end, array $events, array $contacts): array
    {
        $result = [];
        for ($date = new DateTimeImmutable($start); $date < new DateTimeImmutable($end); $date = $date->modify('+1 day')) {
            $day = $date->format('Y-m-d');
            if ($this->select($day, $events, $contacts) === null) {
                $result[] = $day;
            }
        }
        return $result;
    }

    /** @param list<Event> $events @param list<Contact> $contacts */
    public function reminder(string $date, array $events, array $contacts): ?Contact
    {
        $current = $this->select($date, $events, $contacts);
        $previous = $this->select((new DateTimeImmutable($date))->modify('-1 day')->format('Y-m-d'), $events, $contacts);
        return $current !== null && ($previous === null || strcasecmp($current->email, $previous->email) !== 0) ? $current : null;
    }
}
