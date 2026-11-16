<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;

interface DataSource
{
    /** @return list<string> */
    public function admins(): array;
    /** @return list<Contact> */
    public function contacts(): array;
    /**
     * Gibt Ereignisse im Zeitraum von $from inklusive bis $to exklusiv zurück.
     *
     * @return list<Event>
     */
    public function events(DateTimeImmutable $from, DateTimeImmutable $to): array;
    /**
     * Gibt Hinweise auf ungültige Kalendereinträge aus dem letzten events()-Aufruf zurück.
     *
     * @return list<string>
     */
    public function issues(): array;
}
