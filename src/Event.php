<?php

declare(strict_types=1);

namespace Hilfmal;

final readonly class Event
{
    /** @param list<string> $attendees */
    public function __construct(
        public string $uid,
        public string $start,
        public string $end,
        public array $attendees,
        public ?string $created = null,
    ) {
    }
}
