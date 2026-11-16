<?php

declare(strict_types=1);

namespace Hilfmal\Tests;

use DateTimeImmutable;
use Hilfmal\Contact;
use Hilfmal\FilePublisher;
use PHPUnit\Framework\TestCase;

final class FilePublisherTest extends TestCase
{
    public function testPublishedPageExpiresWithoutAnotherRun(): void
    {
        $directory = sys_get_temp_dir() . '/hilfmal-' . bin2hex(random_bytes(6));
        mkdir($directory);
        $path = $directory . '/index.php';
        try {
            $publisher = new FilePublisher($path, '<p>{{name}} {{phone}}</p>');
            $publisher->publish(new Contact('Ada', '123', 'ada@example.test'), new DateTimeImmutable('+1 hour'));
            ob_start();
            include $path;
            $valid = ob_get_clean();
            self::assertStringContainsString('Ada 123', $valid);
            $publisher->publish(new Contact('Ada', '123', 'ada@example.test'), new DateTimeImmutable('-1 second'));
            ob_start();
            include $path;
            $expired = ob_get_clean();
            self::assertStringNotContainsString('123', $expired);
            self::assertStringContainsString('nicht verfügbar', $expired);
        } finally {
            if (is_file($path)) {
                unlink($path);
            }
            rmdir($directory);
        }
    }
}
