<?php

declare(strict_types=1);

namespace Hilfmal;

final readonly class Contact
{
    public function __construct(public string $name, public string $phone, public string $email)
    {
    }
}
