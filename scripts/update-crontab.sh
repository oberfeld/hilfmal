#!/bin/sh
set -eu

cron_job="${1:?Cronjob-Zeile fehlt}"
crontab_command="${CRONTAB_COMMAND:-crontab}"
existing_file="$(mktemp)"
error_file="$(mktemp)"
updated_file="$(mktemp)"
trap 'rm -f "$existing_file" "$error_file" "$updated_file"' 0 HUP INT TERM

if "$crontab_command" -l > "$existing_file" 2> "$error_file"; then
  :
elif grep -Eq '^no crontab for [^[:space:]]+$' "$error_file"; then
  : > "$existing_file"
else
  cat "$error_file" >&2
  exit 1
fi

awk 'index($0, "# hilfmal deployment") == 0' "$existing_file" > "$updated_file"
printf '%s\n' "$cron_job" >> "$updated_file"
"$crontab_command" - < "$updated_file"
