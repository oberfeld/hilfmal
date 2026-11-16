<?php

declare(strict_types=1);

return [
    'dav_user' => 'hilfmal',
    'dav_password' => 'APP-PASSWORT',
    'calendar_url' => 'https://cloud.example.test/remote.php/dav/calendars/hilfmal/dienstplan/',
    'contacts_url' => 'https://cloud.example.test/remote.php/dav/addressbooks/users/hilfmal/mitwirkende/',
    'admins_url' => 'https://cloud.example.test/remote.php/dav/addressbooks/users/hilfmal/administration/',
    'sender' => 'hilfmal@oberfeld.be',
    'emergency_email' => 'admin@example.test',
    'reminder_days' => 7,
    'gap_interval' => 'P1M',
    'public_page' => '/pfad/zum/webverzeichnis/index.php',
    'web_template' => __DIR__ . '/templates/web.html',
    'reminder_template' => __DIR__ . '/templates/reminder.txt',
    'admin_template' => __DIR__ . '/templates/admin.txt',
    'reminder_subject' => 'Erinnerung an deinen Hilfmal-Dienst',
    'admin_subject' => 'Hinweise zum Hilfmal-Dienstplan',
];
