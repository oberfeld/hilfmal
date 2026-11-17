<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;
use DateTimeZone;

/**
 * Wertet Kalendertermine und Kontakte aus, um Dienstzuständigkeit,
 * Erinnerungen, Lücken und Planhinweise zu bestimmen.
 */
final class Planner
{
    /**
     * Gibt den Kalendertag des laufenden Dienstes zurück. Vor 06:00 Uhr
     * Schweizer Zeit gehört der Dienst noch zum vorherigen Tag.
     */
    public function serviceDate(DateTimeImmutable $now): string
    {
        $local = $now->setTimezone(new DateTimeZone('Europe/Zurich'));
        if ((int) $local->format('G') < 6) {
            $local = $local->modify('-1 day');
        }
        return $local->format('Y-m-d');
    }

    /**
     * Wählt den gültigen Kontakt mit der höchsten Priorität für den Diensttag:
     * zuerst kürzere Termine, dann später erstellte Termine und zuletzt die UID.
     * Fehlt einer Dauergruppe ein Erstellungszeitpunkt, entscheidet dort die UID.
     * Pro Termin wird der erste Attendee mit gültigem Kontakt gewählt.
     *
     * @param list<Event> $events
     * @param list<Contact> $contacts
     */
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
        foreach ($event->attendees as $attendee) {
            foreach ($contacts as $contact) {
                if (strcasecmp($contact->email, $attendee) !== 0 || trim($contact->name) === '' || trim($contact->phone) === '' || !filter_var($contact->email, FILTER_VALIDATE_EMAIL)) {
                    continue;
                }
                return $contact;
            }
        }
        return null;
    }

    /**
     * Meldet Einträge, deren eingeladene Person keinem vollständigen,
     * eindeutigen Kontakt zugeordnet werden kann.
     *
     * @param list<Event> $events
     * @param list<Contact> $contacts
     * @return list<string>
     */
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

    /**
     * Gibt Tage ohne gültigen Kontakt zurück; $start ist inklusive, $end exklusiv.
     *
     * @param list<Event> $events
     * @param list<Contact> $contacts
     * @return list<string>
     */
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

    /**
     * Gibt den Kontakt nur am ersten Tag eines zusammenhängenden Dienstes
     * zurück, damit für diesen Dienst nicht täglich erinnert wird.
     *
     * @param list<Event> $events
     * @param list<Contact> $contacts
     */
    public function reminder(string $date, array $events, array $contacts): ?Contact
    {
        $current = $this->select($date, $events, $contacts);
        $previous = $this->select((new DateTimeImmutable($date))->modify('-1 day')->format('Y-m-d'), $events, $contacts);
        return $current !== null && ($previous === null || strcasecmp($current->email, $previous->email) !== 0) ? $current : null;
    }
}
