<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\Application;
use Hilfmal\Contact;
use Hilfmal\DataSource;
use Hilfmal\Event;
use Hilfmal\Mailer;
use Hilfmal\Publisher;
use PHPUnit\Framework\TestCase;
use RuntimeException;

final class ApplicationTest extends TestCase
{
    public function testDailyRunPublishesAndSendsConfiguredRemindersAndGapNotices(): void
    {
        $source = new FakeSource();
        $mailer = new FakeMailer();
        $publisher = new FakePublisher();
        $app = new Application($source, $mailer, $publisher, 'sender@example.test', 'emergency@example.test', 7, 'P1M');
        self::assertSame(0, $app->run(new DateTimeImmutable('2026-11-01 06:00:00 Europe/Zurich')));
        self::assertSame('Aktuell', $publisher->contact?->name);
        self::assertSame('2026-11-02 06:00', $publisher->expires?->format('Y-m-d H:i'));
        self::assertCount(2, $mailer->sent);
        self::assertSame('person@example.test', $mailer->sent[0][0]);
        self::assertSame('sender@example.test', $mailer->sent[0][1]);
        self::assertStringContainsString('2026-11-08', $mailer->sent[0][3]);
        self::assertSame('admin@example.test', $mailer->sent[1][0]);
        self::assertStringContainsString('2026-11-02', $mailer->sent[1][3]);
    }

    public function testUnavailableAdminAddressBookUsesEmergencyAddressAndFails(): void
    {
        $source = new FakeSource();
        $source->failAdmins = true;
        $mailer = new FakeMailer();
        $app = new Application($source, $mailer, new FakePublisher(), 'sender@example.test', 'emergency@example.test', 7, 'P1M');
        self::assertSame(1, $app->run(new DateTimeImmutable('2026-11-01 06:00:00 Europe/Zurich')));
        self::assertSame('emergency@example.test', $mailer->sent[0][0]);
    }

    public function testMailFailureIsNotReportedAsSuccess(): void
    {
        $mailer = new FakeMailer();
        $mailer->fail = true;
        $publisher = new FakePublisher();
        $app = new Application(new FakeSource(), $mailer, $publisher, 'sender@example.test', 'emergency@example.test', 7, 'P1M');
        self::assertSame(1, $app->run(new DateTimeImmutable('2026-11-01 06:00:00 Europe/Zurich')));
        self::assertNull($publisher->expires);
    }

    public function testConfigurableRangesAndTemplate(): void
    {
        $mailer = new FakeMailer();
        $app = new Application(new FakeSource(), $mailer, new FakePublisher(), 'sender@example.test', 'emergency@example.test', 0, 'P2D', 'Dienst {{name}} am {{date}}', 'Fehler: {{issues}}', 'Betreff');
        self::assertSame(0, $app->run(new DateTimeImmutable('2026-11-01 06:00:00 Europe/Zurich')));
        self::assertSame('Betreff', $mailer->sent[0][2]);
        self::assertSame('Dienst Aktuell am 2026-11-01', $mailer->sent[0][3]);
        self::assertStringContainsString('2026-11-02', $mailer->sent[1][3]);
        self::assertStringNotContainsString('2026-11-03', $mailer->sent[1][3]);
    }

    public function testPublisherFailureSendsErrorAndFails(): void
    {
        $publisher = new FakePublisher();
        $publisher->fail = true;
        $mailer = new FakeMailer();
        $app = new Application(new FakeSource(), $mailer, $publisher, 'sender@example.test', 'emergency@example.test', 7, 'P1M');
        self::assertSame(1, $app->run(new DateTimeImmutable('2026-11-01 06:00:00 Europe/Zurich')));
        self::assertSame(['admin@example.test', 'emergency@example.test'], array_column(array_slice($mailer->sent, -2), 0));
    }
}

final class FakeSource implements DataSource
{
    public bool $failAdmins = false;
    public function admins(): array
    {
        if ($this->failAdmins) {
            throw new RuntimeException('Adressbuch nicht verfügbar');
        }
        return ['admin@example.test'];
    }
    public function contacts(): array
    {
        return [new Contact('Aktuell', '123', 'current@example.test'), new Contact('Person', '456', 'person@example.test')];
    }
    public function events(DateTimeImmutable $from, DateTimeImmutable $to): array
    {
        return [new Event('1', '2026-11-01', '2026-11-02', ['current@example.test']), new Event('2', '2026-11-08', '2026-11-09', ['person@example.test'])];
    }
    public function issues(): array
    {
        return [];
    }
}

final class FakeMailer implements Mailer
{
    public array $sent = [];
    public bool $fail = false;
    public function send(string $to, string $from, string $subject, string $body): void
    {
        if ($this->fail) {
            throw new RuntimeException('Versand fehlgeschlagen');
        }
        $this->sent[] = [$to, $from, $subject, $body];
    }
}

final class FakePublisher implements Publisher
{
    public ?Contact $contact = null;
    public ?DateTimeImmutable $expires = null;
    public bool $fail = false;
    public function publish(?Contact $contact, DateTimeImmutable $expires): void
    {
        if ($this->fail) {
            throw new RuntimeException('Schreiben fehlgeschlagen');
        }
        $this->contact = $contact;
        $this->expires = $expires;
    }
}
