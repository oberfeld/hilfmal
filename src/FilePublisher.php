<?php

declare(strict_types=1);

namespace Hilfmal;

use DateTimeImmutable;
use RuntimeException;

final readonly class FilePublisher implements Publisher
{
    public function __construct(private string $path, private string $template)
    {
    }

    /**
     * Erstellt eine PHP-Seite, die beim Abruf selbst prüft, ob die
     * Dienstinformation noch gültig ist. Sie wird über eine temporäre Datei
     * ersetzt, damit keine unvollständig geschriebene Seite ausgeliefert wird.
     */
    public function publish(?Contact $contact, DateTimeImmutable $expires): void
    {
        $page = new WebPage();
        $valid = $page->render($this->template, $contact, $expires, $expires->modify('-1 second'));
        $expired = $page->render($this->template, null, $expires, $expires);
        $content = "<?php\nheader('Cache-Control: no-store, private');\nheader('Content-Type: text/html; charset=UTF-8');\n"
            . 'echo time() < ' . $expires->getTimestamp() . ' ? ' . var_export($valid, true) . ' : ' . var_export($expired, true) . ";\n";
        if (!is_dir(dirname($this->path))) {
            throw new RuntimeException('Webverzeichnis existiert nicht.');
        }
        $temporary = tempnam(dirname($this->path), '.hilfmal-');
        if ($temporary === false) {
            throw new RuntimeException('Temporäre Webdatei konnte nicht angelegt werden.');
        }
        try {
            if (file_put_contents($temporary, $content, LOCK_EX) !== strlen($content) || !chmod($temporary, 0644) || !rename($temporary, $this->path)) {
                throw new RuntimeException('Webseite konnte nicht veröffentlicht werden.');
            }
        } finally {
            if (is_file($temporary)) {
                unlink($temporary);
            }
        }
    }
}
