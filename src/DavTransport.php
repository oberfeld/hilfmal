<?php

declare(strict_types=1);

namespace Hilfmal;

interface DavTransport
{
    /**
     * Sendet eine DAV-REPORT-Anfrage mit XML-Inhalt und gibt die XML-Antwort zurück.
     */
    public function report(string $url, string $body): string;
}
