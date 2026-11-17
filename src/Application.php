<?php

declare(strict_types=1);

namespace Hilfmal;

use DateInterval;
use DateTimeImmutable;
use DateTimeZone;
use Throwable;

final class Application
{
    /**
     * $source, $mailer und $publisher kapseln Datenzugriff, Versand und
     * Veröffentlichung. $emergency ist die Notfall-E-Mail-Adresse;
     * $gapInterval bestimmt den Zeitraum der Lückenprüfung (z. B. «P1M»).
     */
    public function __construct(
        private readonly DataSource $source,
        private readonly Mailer $mailer,
        private readonly Publisher $publisher,
        private readonly string $sender,
        private readonly string $emergency,
        private readonly int $reminderDays,
        private readonly string $gapInterval,
        private readonly string $reminderTemplate = "Hallo {{name}},\n\nDein Pikettdienst beginnt am {{date}}. Organisiere bei Verhinderung bitte eine Vertretung.",
        private readonly string $adminTemplate = "Hinweise zum Dienstplan:\n\n{{issues}}",
        private readonly string $reminderSubject = 'Erinnerung an deinen Hilfmal-Dienst',
        private readonly string $adminSubject = 'Hinweise zum Hilfmal-Dienstplan',
    ) {
        if (!filter_var($sender, FILTER_VALIDATE_EMAIL) || !filter_var($emergency, FILTER_VALIDATE_EMAIL) || $reminderDays < 0) {
            throw new \InvalidArgumentException('Ungültige E-Mail-Adresse oder Erinnerungsfrist.');
        }
        new DateInterval($gapInterval);
    }

    /**
     * Führt einen täglichen Lauf aus und gibt 0 bei Erfolg, sonst 1 zurück.
     * Die Webseite wird erst nach erfolgreichem E-Mail-Versand veröffentlicht;
     * Fehler werden protokolliert und nach Möglichkeit per E-Mail gemeldet.
     */
    public function run(DateTimeImmutable $now): int
    {
        $admins = [];
        $stage = 'Administrationsadressen lesen';
        try {
            $admins = $this->loadAdmins();
            $planner = new Planner();
            [$today, $target, $gapEnd] = $this->calculateDateRange($now, $planner);

            $stage = 'Kalender lesen';
            $events = $this->loadEvents($today, $target, $gapEnd);

            $stage = 'Mitwirkende lesen';
            $contacts = $this->source->contacts();
            $current = $planner->select($today->format('Y-m-d'), $events, $contacts);
            $expires = $today->modify('+1 day')->setTime(6, 0);
            $reminder = $planner->reminder($target->format('Y-m-d'), $events, $contacts);

            if ($reminder !== null) {
                $stage = 'Erinnerung versenden';
                $this->sendReminder($reminder, $target);
            }

            $issues = $this->collectIssues($today, $gapEnd, $planner, $events, $contacts);
            if ($issues !== []) {
                $stage = 'Administrationshinweis versenden';
                $this->sendAdminNotices($admins, $issues);
            }

            $stage = 'Webseite veröffentlichen';
            $this->publisher->publish($current, $expires);
            return 0;
        } catch (Throwable $error) {
            $this->reportFailure($error, $stage, $admins);
            return 1;
        }
    }

    /**
     * @return list<string>
     */
    private function loadAdmins(): array
    {
        $admins = $this->source->admins();
        if ($admins === [] || array_filter($admins, static fn (string $email): bool => !filter_var($email, FILTER_VALIDATE_EMAIL))) {
            throw new \RuntimeException('Administrationsadressbuch enthält keine gültigen Empfänger.');
        }
        return $admins;
    }

    /**
     * @return array{DateTimeImmutable, DateTimeImmutable, DateTimeImmutable}
     */
    private function calculateDateRange(DateTimeImmutable $now, Planner $planner): array
    {
        $today = new DateTimeImmutable($planner->serviceDate($now), new DateTimeZone('Europe/Zurich'));
        $target = $today->modify("+{$this->reminderDays} days");
        $gapEnd = $today->add(new DateInterval($this->gapInterval));
        if ($gapEnd <= $today) {
            throw new \RuntimeException('Der Prüfzeitraum muss grösser als null sein.');
        }
        return [$today, $target, $gapEnd];
    }

    /**
     * @return list<Event>
     */
    private function loadEvents(DateTimeImmutable $today, DateTimeImmutable $target, DateTimeImmutable $gapEnd): array
    {
        $end = max($gapEnd, $target->modify('+1 day'));
        return $this->source->events($today->modify('-1 day'), $end);
    }

    private function sendReminder(Contact $contact, DateTimeImmutable $target): void
    {
        $body = strtr($this->reminderTemplate, [
            '{{name}}' => $contact->name,
            '{{date}}' => $target->format('Y-m-d'),
        ]);
        $this->mailer->send($contact->email, $this->sender, $this->reminderSubject, $body);
    }

    /**
     * @param list<Event> $events
     * @param list<Contact> $contacts
     * @return list<string>
     */
    private function collectIssues(DateTimeImmutable $today, DateTimeImmutable $gapEnd, Planner $planner, array $events, array $contacts): array
    {
        $issues = [...$this->source->issues(), ...$planner->issues($events, $contacts)];
        foreach ($planner->gaps($today->format('Y-m-d'), $gapEnd->format('Y-m-d'), $events, $contacts) as $gap) {
            $issues[] = "Kein gültiger Dienst: $gap";
        }
        return $issues;
    }

    /**
     * @param list<string> $admins
     * @param list<string> $issues
     */
    private function sendAdminNotices(array $admins, array $issues): void
    {
        $body = strtr($this->adminTemplate, ['{{issues}}' => implode("\n", $issues)]);
        foreach (array_unique($admins) as $admin) {
            $this->mailer->send($admin, $this->sender, $this->adminSubject, $body);
        }
    }

    /**
     * @param list<string> $admins
     */
    private function reportFailure(Throwable $error, string $stage, array $admins): void
    {
        error_log('Hilfmal: Programmlauf fehlgeschlagen beim Schritt "' . $stage . '": ' . get_class($error));
        $recipients = $admins !== [] ? [...$admins, $this->emergency] : [$this->emergency];
        foreach (array_unique($recipients) as $admin) {
            try {
                $this->mailer->send($admin, $this->sender, 'Fehler beim Hilfmal-Programmlauf', 'Der Hilfmal-Programmlauf ist fehlgeschlagen. Bitte das Serverprotokoll und die Konfiguration prüfen.');
            } catch (Throwable $mailError) {
                error_log('Hilfmal: Versand der Fehlermeldung fehlgeschlagen: ' . get_class($mailError));
            }
        }
    }
}
