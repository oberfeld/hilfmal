<?php

declare(strict_types=1);

/**
 * Kommandozeilen-Einstiegspunkt für den täglichen Hilfmal-Lauf.
 *
 * Aufruf: php bin/hilfmal.php /pfad/zur/konfiguration.php
 *
 * Lädt und prüft die Konfiguration, verbindet Datenzugriff, E-Mail-Versand
 * und Veröffentlichung und startet die Anwendung mit der aktuellen Systemzeit.
 * Der Cronjob verwendet denselben Einstiegspunkt. Ein Exit-Code von 0 bedeutet
 * Erfolg, 1 einen Fehler; Startfehler werden protokolliert und nach Möglichkeit
 * an die konfigurierte Notfalladresse gemeldet.
 */

use Hilfmal\Application;
use Hilfmal\CurlDavTransport;
use Hilfmal\FilePublisher;
use Hilfmal\NextcloudSource;
use Hilfmal\PhpMailer;

require __DIR__ . '/../vendor/autoload.php';

$config = null;
try {
    if ($argc !== 2 || !is_file($argv[1])) {
        throw new RuntimeException('Aufruf: php bin/hilfmal.php /pfad/zur/konfiguration.php');
    }
    $config = require $argv[1];
    if (!is_array($config)) {
        throw new RuntimeException('Die Konfiguration muss ein Array zurückgeben.');
    }
    $strings = ['dav_user', 'dav_password', 'calendar_url', 'contacts_url', 'admins_url', 'sender', 'emergency_email', 'gap_interval', 'public_page', 'web_template', 'reminder_template', 'admin_template', 'reminder_subject', 'admin_subject'];
    foreach ($strings as $key) {
        if (!isset($config[$key]) || !is_string($config[$key]) || trim($config[$key]) === '') {
            throw new RuntimeException("Konfiguration fehlt oder ist ungültig: $key");
        }
    }
    if (!isset($config['reminder_days']) || !is_int($config['reminder_days'])) {
        throw new RuntimeException('Konfiguration fehlt oder ist ungültig: reminder_days');
    }
    $readTemplate = static function (string $path): string {
        $content = file_get_contents($path);
        if ($content === false) {
            throw new RuntimeException('Template konnte nicht gelesen werden.');
        }
        return $content;
    };
    $source = new NextcloudSource(new CurlDavTransport($config['dav_user'], $config['dav_password']), $config['calendar_url'], $config['contacts_url'], $config['admins_url']);
    $app = new Application(
        $source,
        new PhpMailer(),
        new FilePublisher($config['public_page'], $readTemplate($config['web_template'])),
        $config['sender'],
        $config['emergency_email'],
        $config['reminder_days'],
        $config['gap_interval'],
        $readTemplate($config['reminder_template']),
        $readTemplate($config['admin_template']),
        $config['reminder_subject'],
        $config['admin_subject'],
    );
    exit($app->run(new DateTimeImmutable()));
} catch (Throwable $error) {
    error_log('Hilfmal: Start fehlgeschlagen: ' . $error->getMessage());
    if (is_array($config) && isset($config['sender'], $config['emergency_email']) && is_string($config['sender']) && is_string($config['emergency_email'])) {
        try {
            (new PhpMailer())->send($config['emergency_email'], $config['sender'], 'Fehler beim Hilfmal-Programmstart', 'Der Hilfmal-Programmlauf konnte nicht gestartet werden. Bitte das Serverprotokoll und die Konfiguration prüfen.');
        } catch (Throwable $mailError) {
            error_log('Hilfmal: Versand der Fehlermeldung fehlgeschlagen: ' . get_class($mailError));
        }
    }
    exit(1);
}
