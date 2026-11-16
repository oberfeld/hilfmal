<?php

declare(strict_types=1);

namespace Hilfmal;

use RuntimeException;

final readonly class CurlDavTransport implements DavTransport
{
    public function __construct(private string $user, private string $password)
    {
    }

    public function report(string $url, string $body): string
    {
        if (parse_url($url, PHP_URL_SCHEME) !== 'https') {
            throw new RuntimeException('DAV-Zugriff erfordert HTTPS.');
        }
        $curl = curl_init($url);
        if ($curl === false) {
            throw new RuntimeException('DAV-Verbindung konnte nicht initialisiert werden.');
        }
        try {
            curl_setopt_array($curl, [
                CURLOPT_CUSTOMREQUEST => 'REPORT',
                CURLOPT_USERPWD => $this->user . ':' . $this->password,
                CURLOPT_HTTPAUTH => CURLAUTH_BASIC,
                CURLOPT_HTTPHEADER => ['Depth: 1', 'Content-Type: application/xml; charset=utf-8'],
                CURLOPT_POSTFIELDS => $body,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_CONNECTTIMEOUT => 10,
                CURLOPT_TIMEOUT => 60,
                CURLOPT_FOLLOWLOCATION => false,
                CURLOPT_PROTOCOLS => CURLPROTO_HTTPS,
            ]);
            $response = curl_exec($curl);
            if ($response === false || curl_getinfo($curl, CURLINFO_RESPONSE_CODE) !== 207) {
                throw new RuntimeException('DAV-Anfrage fehlgeschlagen (HTTP ' . curl_getinfo($curl, CURLINFO_RESPONSE_CODE) . ').');
            }
            return $response;
        } finally {
            curl_close($curl);
        }
    }
}
