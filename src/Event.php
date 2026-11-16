<?php

declare(strict_types=1);

namespace Hilfmal;

final readonly class Event
{
    /**
     * $start und $end sind Kalenderdaten im Format JJJJ-MM-TT; $end ist exklusiv.
     * $attendees enthält die E-Mail-Adressen der eingeladenen Personen.
     *
     * @param list<string> $attendees
     */
    public function __construct(
        public string $uid,
        public string $start,
        public string $end,
        public array $attendees,
        public ?string $created = null,
    ) {
    }
}
