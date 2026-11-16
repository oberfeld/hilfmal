<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;

final class WebPage
{
    public function render(string $template, ?Contact $contact, DateTimeImmutable $expires, DateTimeImmutable $now): string
    {
        if ($contact === null || $now >= $expires) {
            return '<p>Eine verlässliche Dienstinformation ist zurzeit nicht verfügbar.</p>';
        }
        return strtr($template, [
            '{{name}}' => htmlspecialchars($contact->name, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8'),
            '{{phone}}' => htmlspecialchars($contact->phone, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8'),
        ]);
    }
}
