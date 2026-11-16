<?php

declare(strict_types=1);

use Hilfmal\Application;
use Hilfmal\CurlDavTransport;
use Hilfmal\FilePublisher;
use Hilfmal\NextcloudSource;
use Hilfmal\PhpMailer;
use Hilfmal\Planner;

if (getenv('HILFMAL_INTEGRATION_TEST') !== '1') {
    throw new RuntimeException('Integrationstest darf nur in der isolierten Testumgebung gestartet werden.');
}

require __DIR__ . '/../../vendor/autoload.php';

const CLOUD = 'https://nextcloud.test';
const MAIL = 'http://mailpit:8025';
const USER = 'hilfmal';
const PASSWORD = 'integration-test-only';

function request(string $method, string $url, ?string $body = null, array $headers = [], bool $auth = false): array
{
    $curl = curl_init($url);
    if ($curl === false) {
        throw new RuntimeException('HTTP-Client konnte nicht initialisiert werden.');
    }
    try {
        curl_setopt_array($curl, [
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CONNECTTIMEOUT => 3,
            CURLOPT_TIMEOUT => 15,
            CURLOPT_FOLLOWLOCATION => false,
            CURLOPT_HTTPHEADER => $headers,
        ]);
        if ($body !== null) {
            curl_setopt($curl, CURLOPT_POSTFIELDS, $body);
        }
        if ($auth) {
            curl_setopt($curl, CURLOPT_USERPWD, USER . ':' . PASSWORD);
        }
        $result = curl_exec($curl);
        if ($result === false) {
            throw new RuntimeException("HTTP-Anfrage fehlgeschlagen: " . curl_error($curl));
        }
        return [curl_getinfo($curl, CURLINFO_RESPONSE_CODE), $result];
    } finally {
        curl_close($curl);
    }
}

function check(bool $condition, string $message): void
{
    if (!$condition) {
        throw new RuntimeException($message);
    }
}

function waitFor(callable $ready, string $name): void
{
    $deadline = microtime(true) + 180;
    do {
        try {
            if ($ready()) {
                return;
            }
        } catch (RuntimeException) {
            // Services may not yet be accepting connections.
        }
        usleep(500000);
    } while (microtime(true) < $deadline);
    throw new RuntimeException("$name ist nach 180 Sekunden nicht bereit.");
}

function dav(string $method, string $path, string $body, string $contentType, array $expected): void
{
    [$status] = request($method, CLOUD . $path, $body, ["Content-Type: $contentType"], true);
    check(in_array($status, $expected, true), "$method $path fehlgeschlagen (HTTP $status).");
}

function card(string $uid, string $name, string $phone, string $email): string
{
    return "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:$uid\r\nN:Test;$name;;;\r\nFN:$name Test\r\nTEL:$phone\r\nEMAIL:$email\r\nEND:VCARD\r\n";
}

function event(string $uid, DateTimeImmutable $start, DateTimeImmutable $end, string $email): string
{
    return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Hilfmal integration test//EN\r\n"
        . "BEGIN:VEVENT\r\nUID:$uid\r\nDTSTAMP:20260101T000000Z\r\nCREATED:20260101T000000Z\r\n"
        . 'DTSTART;VALUE=DATE:' . $start->format('Ymd') . "\r\n"
        . 'DTEND;VALUE=DATE:' . $end->format('Ymd') . "\r\n"
        . "SUMMARY:Testdienst\r\nATTENDEE;PARTSTAT=DECLINED:mailto:$email\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n";
}

/** @return list<array<string, mixed>> */
function messages(): array
{
    [$status, $body] = request('GET', MAIL . '/api/v1/messages');
    check($status === 200, 'Mailpit-Nachrichten konnten nicht gelesen werden.');
    $data = json_decode($body, true, flags: JSON_THROW_ON_ERROR);
    return $data['messages'] ?? [];
}

function messageFor(string $recipient, string $content): void
{
    foreach (messages() as $message) {
        foreach ($message['To'] ?? [] as $to) {
            if (($to['Address'] ?? '') !== $recipient) {
                continue;
            }
            [$status, $body] = request('GET', MAIL . '/api/v1/message/' . rawurlencode($message['ID']));
            check($status === 200, 'Mailpit-Nachricht konnte nicht gelesen werden.');
            $data = json_decode($body, true, flags: JSON_THROW_ON_ERROR);
            if (str_contains($data['Text'] ?? '', $content)) {
                check(($data['From']['Address'] ?? '') === 'hilfmal@example.test', 'Unerwarteter E-Mail-Absender.');
                return;
            }
        }
    }
    throw new RuntimeException("Erwartete E-Mail an $recipient mit '$content' fehlt.");
}

function clearMessages(): void
{
    [$status] = request('DELETE', MAIL . '/api/v1/messages');
    check($status >= 200 && $status < 300, 'Mailpit konnte nicht geleert werden.');
}

function app(NextcloudSource $source, string $page): Application
{
    return new Application(
        $source,
        new PhpMailer(),
        new FilePublisher($page, file_get_contents('/app/templates/web.html')),
        'hilfmal@example.test',
        'notfall@example.test',
        7,
        'P1M',
        file_get_contents('/app/templates/reminder.txt'),
        file_get_contents('/app/templates/admin.txt'),
    );
}

function page(): string
{
    [$status, $body] = request('GET', 'http://127.0.0.1:8080/index.php');
    check($status === 200, "Webseite nicht erreichbar (HTTP $status).");
    return $body;
}

waitFor(static function (): bool {
    [$status, $body] = request('GET', CLOUD . '/status.php');
    $data = json_decode($body, true);
    return $status === 200 && ($data['installed'] ?? false) === true;
}, 'Nextcloud');
waitFor(static fn (): bool => request('GET', MAIL . '/api/v1/info')[0] === 200, 'Mailpit');

$addressBase = '/remote.php/dav/addressbooks/users/hilfmal/';
$calendarBase = '/remote.php/dav/calendars/hilfmal/';
$collection = '<?xml version="1.0"?><d:mkcol xmlns:d="DAV:" xmlns:a="urn:ietf:params:xml:ns:carddav"><d:set><d:prop><d:resourcetype><d:collection/><a:addressbook/></d:resourcetype></d:prop></d:set></d:mkcol>';
foreach (['mitwirkende', 'administration'] as $name) {
    dav('MKCOL', $addressBase . $name . '/', $collection, 'application/xml; charset=utf-8', [201]);
}
dav('MKCALENDAR', $calendarBase . 'dienstplan/', '', 'application/xml; charset=utf-8', [201]);

$contactPath = $addressBase . 'mitwirkende/';
dav('PUT', $contactPath . 'ada.vcf', card('ada', 'Ada', '111', 'ada@example.test'), 'text/vcard', [201, 204]);
dav('PUT', $contactPath . 'bea.vcf', card('bea', 'Bea', '222', 'bea@example.test'), 'text/vcard', [201, 204]);
dav('PUT', $contactPath . 'cara.vcf', card('cara', 'Cara', '333', 'cara@example.test'), 'text/vcard', [201, 204]);
dav('PUT', $addressBase . 'administration/admin.vcf', card('admin', 'Admin', '444', 'admin@example.test'), 'text/vcard', [201, 204]);

$now = new DateTimeImmutable('now', new DateTimeZone('Europe/Zurich'));
$day = new DateTimeImmutable((new Planner())->serviceDate($now), new DateTimeZone('Europe/Zurich'));
$fixedTime = $day->setTime(6, 0);
$calendarPath = $calendarBase . 'dienstplan/';
dav('PUT', $calendarPath . 'base.ics', event('base', $day->modify('-1 day'), $day->modify('+3 days'), 'ada@example.test'), 'text/calendar', [201, 204]);
dav('PUT', $calendarPath . 'cover.ics', event('cover', $day, $day->modify('+1 day'), 'bea@example.test'), 'text/calendar', [201, 204]);
dav('PUT', $calendarPath . 'future.ics', event('future', $day->modify('+7 days'), $day->modify('+9 days'), 'cara@example.test'), 'text/calendar', [201, 204]);

$public = sys_get_temp_dir() . '/hilfmal-public';
check(mkdir($public, 0755), 'Test-Webverzeichnis konnte nicht erstellt werden.');
$pagePath = $public . '/index.php';
$server = proc_open(['php', '-S', '127.0.0.1:8080', '-t', $public], [0 => ['pipe', 'r'], 1 => ['pipe', 'w'], 2 => ['pipe', 'w']], $pipes);
check(is_resource($server), 'Test-Webserver konnte nicht gestartet werden.');
fclose($pipes[0]);
try {
    waitFor(static fn (): bool => request('GET', 'http://127.0.0.1:8080/')[0] > 0, 'Test-Webserver');
    $source = new NextcloudSource(new CurlDavTransport(USER, PASSWORD), CLOUD . $calendarPath, CLOUD . $contactPath, CLOUD . $addressBase . 'administration/');
    clearMessages();
    check(app($source, $pagePath)->run($fixedTime) === 0, 'Erster Programmlauf fehlgeschlagen.');
    $first = page();
    check(str_contains($first, 'Bea') && str_contains($first, '222') && !str_contains($first, 'Ada'), 'Vertretung wird nicht korrekt angezeigt.');
    check(!str_contains($first, '@example.test'), 'E-Mail-Adresse ist öffentlich sichtbar.');
    messageFor('cara@example.test', $day->modify('+7 days')->format('Y-m-d'));
    messageFor('admin@example.test', $day->modify('+3 days')->format('Y-m-d'));

    dav('PUT', $contactPath . 'bea.vcf', card('bea', 'Bea', '999', 'bea@example.test'), 'text/vcard', [200, 204]);
    check(str_contains(page(), '222') && !str_contains(page(), '999'), 'Änderung wurde vor dem nächsten Lauf sichtbar.');
    clearMessages();
    check(app($source, $pagePath)->run($fixedTime) === 0, 'Zweiter Programmlauf fehlgeschlagen.');
    check(str_contains(page(), '999') && !str_contains(page(), '222'), 'Änderung fehlt nach erneutem Lauf.');

    $config = tempnam(sys_get_temp_dir(), 'hilfmal-config-');
    check($config !== false, 'Temporäre Konfiguration konnte nicht erstellt werden.');
    try {
        $settings = [
            'dav_user' => USER,
            'dav_password' => PASSWORD,
            'calendar_url' => CLOUD . $calendarPath,
            'contacts_url' => CLOUD . $contactPath,
            'admins_url' => CLOUD . $addressBase . 'administration/',
            'sender' => 'hilfmal@example.test',
            'emergency_email' => 'notfall@example.test',
            'reminder_days' => 7,
            'gap_interval' => 'P1M',
            'public_page' => $pagePath,
            'web_template' => '/app/templates/web.html',
            'reminder_template' => '/app/templates/reminder.txt',
            'admin_template' => '/app/templates/admin.txt',
            'reminder_subject' => 'Erinnerung',
            'admin_subject' => 'Dienstplan',
        ];
        check(file_put_contents($config, '<?php return ' . var_export($settings, true) . ';') !== false, 'Testkonfiguration konnte nicht geschrieben werden.');
        clearMessages();
        $process = proc_open(['php', '/app/bin/hilfmal.php', $config], [0 => ['pipe', 'r'], 1 => ['pipe', 'w'], 2 => ['pipe', 'w']], $cliPipes);
        check(is_resource($process), 'Programm konnte nicht gestartet werden.');
        fclose($cliPipes[0]);
        $stdout = stream_get_contents($cliPipes[1]);
        $stderr = stream_get_contents($cliPipes[2]);
        fclose($cliPipes[1]);
        fclose($cliPipes[2]);
        check(proc_close($process) === 0, "Aufruf von bin/hilfmal.php fehlgeschlagen: $stdout $stderr");
        check(str_contains(page(), '999'), 'Produktiver Programmeinstieg veröffentlichte nicht den erwarteten Kontakt.');
        messageFor('cara@example.test', $day->modify('+7 days')->format('Y-m-d'));
    } finally {
        unlink($config);
    }

    clearMessages();
    $broken = new NextcloudSource(new CurlDavTransport(USER, PASSWORD), CLOUD . $calendarBase . 'nicht-vorhanden/', CLOUD . $contactPath, CLOUD . $addressBase . 'administration/');
    check(app($broken, $pagePath)->run($fixedTime) === 1, 'Fehlerhafter Datenzugriff muss fehlschlagen.');
    messageFor('admin@example.test', 'fehlgeschlagen');
    messageFor('notfall@example.test', 'fehlgeschlagen');
    check(str_contains(page(), '999'), 'Fehlgeschlagener Lauf hat die bestehende Seite verändert.');

    clearMessages();
    $missingAdmins = new NextcloudSource(new CurlDavTransport(USER, PASSWORD), CLOUD . $calendarPath, CLOUD . $contactPath, CLOUD . $addressBase . 'nicht-vorhanden/');
    check(app($missingAdmins, $pagePath)->run($fixedTime) === 1, 'Fehlerhafter Zugriff auf das Administrationsadressbuch muss fehlschlagen.');
    messageFor('notfall@example.test', 'fehlgeschlagen');
    check(str_contains(page(), '999'), 'Fehlgeschlagener Lauf hat die bestehende Seite verändert.');

    $publisher = new FilePublisher($pagePath, file_get_contents('/app/templates/web.html'));
    $publisher->publish(new Hilfmal\Contact('Abgelaufen', '777', 'expired@example.test'), $now->modify('-1 second'));
    $expired = page();
    check(!str_contains($expired, '777') && str_contains($expired, 'nicht verfügbar'), 'Abgelaufene Information wird weiterhin angezeigt.');
    echo "Integrationstests erfolgreich: Nextcloud, E-Mail, Webseite und Fehlerbehandlung.\n";
} finally {
    proc_terminate($server);
    fclose($pipes[1]);
    fclose($pipes[2]);
    proc_close($server);
    if (is_file($pagePath)) {
        unlink($pagePath);
    }
    rmdir($public);
}
