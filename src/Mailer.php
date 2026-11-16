<?php

declare(strict_types=1);

namespace Hilfmal;

interface Mailer
{
    public function send(string $to, string $from, string $subject, string $body): void;
}
