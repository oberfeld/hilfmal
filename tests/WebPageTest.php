<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\Contact;
use Hilfmal\WebPage;
use PHPUnit\Framework\TestCase;

final class WebPageTest extends TestCase
{
    public function testEscapingAndExpiry(): void
    {
        $page = new WebPage();
        $template = '<p>{{name}} {{phone}}</p>';
        $contact = new Contact('<script>', '123 & 456', 'private@example.test');
        $expires = new DateTimeImmutable('2026-11-02 06:00:00 Europe/Zurich');
        $valid = $page->render($template, $contact, $expires, $expires->modify('-1 second'));
        self::assertStringContainsString('&lt;script&gt;', $valid);
        self::assertStringContainsString('123 &amp; 456', $valid);
        self::assertStringNotContainsString('private@example.test', $valid);
        $expired = $page->render($template, $contact, $expires, $expires);
        self::assertStringNotContainsString('123', $expired);
        self::assertStringContainsString('nicht verfügbar', $expired);
    }
}
