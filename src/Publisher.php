<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;

interface Publisher
{
    public function publish(?Contact $contact, DateTimeImmutable $expires): void;
}
