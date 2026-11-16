<?php

declare(strict_types=1);

namespace Hilfmal;

interface DavTransport
{
    public function report(string $url, string $body): string;
}
