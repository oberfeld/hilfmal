#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"
project="hilfmal-it-$(date +%s)-$$"

cleanup() {
    docker compose -p "$project" -f tests/integration/compose.yaml down --volumes --remove-orphans
}
trap cleanup EXIT
docker compose -p "$project" -f tests/integration/compose.yaml up --build --abort-on-container-exit --exit-code-from runner
