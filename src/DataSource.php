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
    /** @return list<Event> */
    public function events(DateTimeImmutable $from, DateTimeImmutable $to): array;
    /** @return list<string> */
    public function issues(): array;
}
