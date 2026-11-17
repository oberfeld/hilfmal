FROM golang:1.27.4-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY pkg ./pkg
COPY templates ./templates
RUN mkdir -p /build \
    && go build -o /build/hilfmal ./cmd/hilfmal \
    && go build -o /build/integration ./cmd/integration

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates msmtp openssl socat \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /build/hilfmal /app/hilfmal
COPY --from=build /build/integration /app/integration
COPY templates ./templates

RUN openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
    -keyout /etc/ssl/private/hilfmal-test.key -out /usr/local/share/ca-certificates/hilfmal-test.crt \
    -subj '/CN=nextcloud.test' -addext 'subjectAltName=DNS:nextcloud.test' \
    && update-ca-certificates \
    && printf '%s\n' '#!/bin/sh' 'exec /usr/bin/msmtp -t -i --read-envelope-from --host=mailpit --port=1025 --tls=off "$@"' > /usr/local/bin/sendmail \
    && chmod 0755 /usr/local/bin/sendmail
