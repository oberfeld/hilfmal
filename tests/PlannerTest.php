<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\Contact;
use Hilfmal\Event;
use Hilfmal\Planner;
use PHPUnit\Framework\TestCase;

final class PlannerTest extends TestCase
{
    private function event(string $uid, string $start, string $end, string $email, ?string $created = null): Event
    {
        return new Event($uid, $start, $end, [$email], $created);
    }

    public function testServiceDateChangesAtSixInZurichIncludingDst(): void
    {
        $planner = new Planner();
        foreach (['2026-03-29', '2026-10-25', '2026-01-01'] as $date) {
            self::assertSame((new DateTimeImmutable($date))->modify('-1 day')->format('Y-m-d'), $planner->serviceDate(new DateTimeImmutable("$date 05:59:59 Europe/Zurich")));
            self::assertSame($date, $planner->serviceDate(new DateTimeImmutable("$date 06:00:00 Europe/Zurich")));
            self::assertSame($date, $planner->serviceDate(new DateTimeImmutable("$date 06:00:01 Europe/Zurich")));
        }
    }

    public function testPriorityAndFallbackAreIndependentOfInputOrder(): void
    {
        $events = [
            $this->event('base', '2026-11-01', '2026-11-10', 'base@example.test'),
            $this->event('z', '2026-11-03', '2026-11-05', 'z@example.test', '2026-01-02T00:00:00Z'),
            $this->event('a', '2026-11-03', '2026-11-05', 'a@example.test', '2026-01-02T00:00:00Z'),
            $this->event('old', '2026-11-03', '2026-11-05', 'old@example.test', '2026-01-01T00:00:00Z'),
            $this->event('invalid', '2026-11-04', '2026-11-05', 'missing@example.test'),
        ];
        $contacts = [
            new Contact('Base', '111', 'base@example.test'),
            new Contact('A', '222', 'a@example.test'),
            new Contact('Z', '333', 'z@example.test'),
            new Contact('Old', '444', 'old@example.test'),
        ];
        $planner = new Planner();
        foreach ([$events, array_reverse($events)] as $order) {
            self::assertSame('A', $planner->select('2026-11-04', $order, $contacts)?->name);
            self::assertNotEmpty($planner->issues($order, $contacts));
        }
    }

    public function testMissingOrAmbiguousContactNeverProducesNumber(): void
    {
        $planner = new Planner();
        $event = $this->event('1', '2026-11-01', '2026-11-02', 'a@example.test');
        self::assertNull($planner->select('2026-11-01', [$event], []));
        self::assertNull($planner->select('2026-11-01', [$event], [new Contact('A', '1', 'a@example.test'), new Contact('B', '2', 'a@example.test')]));
        self::assertNull($planner->select('2026-11-01', [$event], [new Contact('A', '', 'a@example.test')]));
    }

    public function testLaterCreationAndMissingCreationTiebreak(): void
    {
        $planner = new Planner();
        $contacts = [new Contact('A', '1', 'a@example.test'), new Contact('B', '2', 'b@example.test'), new Contact('C', '3', 'c@example.test')];
        $a = $this->event('a', '2026-11-01', '2026-11-02', 'a@example.test', '2026-01-01T00:00:00Z');
        $b = $this->event('b', '2026-11-01', '2026-11-02', 'b@example.test', '2026-01-02T00:00:00Z');
        $c = $this->event('c', '2026-11-01', '2026-11-02', 'c@example.test');
        self::assertSame('B', $planner->select('2026-11-01', [$a, $b], $contacts)?->name);
        foreach ([[$a, $b, $c], [$c, $b, $a]] as $events) {
            self::assertSame('A', $planner->select('2026-11-01', $events, $contacts)?->name);
        }
    }

    public function testGapsAndSingleReminderAtStartOfContiguousAssignment(): void
    {
        $planner = new Planner();
        $events = [$this->event('one', '2026-12-01', '2026-12-04', 'a@example.test')];
        $contacts = [new Contact('A', '123', 'a@example.test')];
        self::assertSame(['2026-12-04'], $planner->gaps('2026-12-01', '2026-12-05', $events, $contacts));
        self::assertSame('A', $planner->reminder('2026-12-01', $events, $contacts)?->name);
        self::assertNull($planner->reminder('2026-12-02', $events, $contacts));
    }

    public function testOverlapChangesReminderRecipientOnlyAtStartOfTheirDuty(): void
    {
        $planner = new Planner();
        $events = [
            $this->event('base', '2026-12-01', '2026-12-08', 'a@example.test'),
            $this->event('cover', '2026-12-03', '2026-12-05', 'b@example.test'),
        ];
        $contacts = [new Contact('A', '1', 'a@example.test'), new Contact('B', '2', 'b@example.test')];
        self::assertSame('B', $planner->reminder('2026-12-03', $events, $contacts)?->name);
        self::assertNull($planner->reminder('2026-12-04', $events, $contacts));
        self::assertSame('A', $planner->reminder('2026-12-05', $events, $contacts)?->name);
    }
}
