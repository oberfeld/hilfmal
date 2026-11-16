<?php

declare(strict_types=1);

namespace Hilfmal;

use RuntimeException;

final class PhpMailer implements Mailer
{
    /**
     * Übergibt die Nachricht an den lokalen Mailversand. Dessen Annahme
     * bestätigt noch nicht die Zustellung an den Empfänger.
     */
    public function send(string $to, string $from, string $subject, string $body): void
    {
        if (!filter_var($to, FILTER_VALIDATE_EMAIL) || !filter_var($from, FILTER_VALIDATE_EMAIL) || str_contains($subject, "\n") || str_contains($subject, "\r")) {
            throw new RuntimeException('Ungültige E-Mail-Angaben.');
        }
        $encoded = mb_encode_mimeheader($subject, 'UTF-8', 'B', "\r\n");
        if (!mail($to, $encoded, $body, ['From' => $from, 'Reply-To' => $from, 'Content-Type' => 'text/plain; charset=UTF-8'])) {
            throw new RuntimeException('Mailversand fehlgeschlagen.');
        }
    }
}
