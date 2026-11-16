<?php

declare(strict_types=1);

namespace Hilfmal;

use DateInterval;
use DateTimeImmutable;
use DateTimeZone;
use Throwable;

final class Application
{
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

    public function run(DateTimeImmutable $now): int
    {
        $admins = [];
        $stage = 'Administrationsadressen lesen';
        try {
            $admins = $this->source->admins();
            if ($admins === [] || array_filter($admins, static fn (string $email): bool => !filter_var($email, FILTER_VALIDATE_EMAIL))) {
                throw new \RuntimeException('Administrationsadressbuch enthält keine gültigen Empfänger.');
            }
            $planner = new Planner();
            $today = new DateTimeImmutable($planner->serviceDate($now), new DateTimeZone('Europe/Zurich'));
            $target = $today->modify("+{$this->reminderDays} days");
            $gapEnd = $today->add(new DateInterval($this->gapInterval));
            if ($gapEnd <= $today) {
                throw new \RuntimeException('Der Prüfzeitraum muss grösser als null sein.');
            }
            $end = max($gapEnd, $target->modify('+1 day'));
            $stage = 'Kalender lesen';
            $events = $this->source->events($today->modify('-1 day'), $end);
            $stage = 'Mitwirkende lesen';
            $contacts = $this->source->contacts();
            $current = $planner->select($today->format('Y-m-d'), $events, $contacts);
            $expires = $today->modify('+1 day')->setTime(6, 0);
            $reminder = $planner->reminder($target->format('Y-m-d'), $events, $contacts);
            if ($reminder !== null) {
                $stage = 'Erinnerung versenden';
                $this->mailer->send($reminder->email, $this->sender, $this->reminderSubject, strtr($this->reminderTemplate, ['{{name}}' => $reminder->name, '{{date}}' => $target->format('Y-m-d')]));
            }
            $gaps = $planner->gaps($today->format('Y-m-d'), $gapEnd->format('Y-m-d'), $events, $contacts);
            $issues = [...$this->source->issues(), ...$planner->issues($events, $contacts)];
            foreach ($gaps as $gap) {
                $issues[] = "Kein gültiger Dienst: $gap";
            }
            if ($issues !== []) {
                foreach (array_unique($admins) as $admin) {
                    $stage = 'Administrationshinweis versenden';
                    $this->mailer->send($admin, $this->sender, $this->adminSubject, strtr($this->adminTemplate, ['{{issues}}' => implode("\n", $issues)]));
                }
            }
            $stage = 'Webseite veröffentlichen';
            $this->publisher->publish($current, $expires);
            return 0;
        } catch (Throwable $error) {
            error_log('Hilfmal: Programmlauf fehlgeschlagen beim Schritt "' . $stage . '": ' . get_class($error));
            foreach (array_unique($admins !== [] ? [...$admins, $this->emergency] : [$this->emergency]) as $admin) {
                try {
                    $this->mailer->send($admin, $this->sender, 'Fehler beim Hilfmal-Programmlauf', 'Der Hilfmal-Programmlauf ist fehlgeschlagen. Bitte das Serverprotokoll und die Konfiguration prüfen.');
                } catch (Throwable $mailError) {
                    error_log('Hilfmal: Versand der Fehlermeldung fehlgeschlagen: ' . get_class($mailError));
                }
            }
            return 1;
        }
    }
}
