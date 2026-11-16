FROM composer:2.8.12 AS composer

FROM php:8.3-cli-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends git unzip libcurl4-openssl-dev libonig-dev libxml2-dev msmtp socat openssl \
    && docker-php-ext-install curl mbstring \
    && rm -rf /var/lib/apt/lists/*

COPY --from=composer /usr/bin/composer /usr/local/bin/composer
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-interaction --no-progress --no-scripts --no-autoloader
COPY src ./src
RUN composer dump-autoload --no-dev --no-interaction
COPY bin ./bin
COPY templates ./templates
COPY tests/integration ./tests/integration

RUN openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
    -keyout /etc/ssl/private/hilfmal-test.key -out /usr/local/share/ca-certificates/hilfmal-test.crt \
    -subj '/CN=nextcloud.test' -addext 'subjectAltName=DNS:nextcloud.test' \
    && update-ca-certificates \
    && printf '%s\n' 'sendmail_path = "/usr/bin/msmtp -t -i --read-envelope-from --host=mailpit --port=1025 --tls=off"' > /usr/local/etc/php/conf.d/hilfmal-test.ini
