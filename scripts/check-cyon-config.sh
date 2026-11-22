#!/usr/bin/env bash
set -euo pipefail

config_file="${1:?Konfigurationsdatei fehlt}"
mode="${2:-check}"

read_value() {
  local key="$1" line found=false value=""
  while IFS= read -r line || [ -n "$line" ]; do
    if [[ "$line" =~ ^[[:space:]]*$key[[:space:]]*: ]]; then
      if [ "$found" = true ] || ! [[ "$line" =~ ^$key:[[:space:]]*\"([^\"]+)\"[[:space:]]*(#.*)?$ ]]; then
        echo "Ungültiger oder mehrfacher Eintrag in $config_file: $key" >&2
        return 1
      fi
      found=true
      value="${BASH_REMATCH[1]}"
    fi
  done < "$config_file"
  if [ "$found" = false ]; then
    echo "Fehlender Eintrag in $config_file: $key" >&2
    return 1
  fi
  printf '%s\n' "$value"
}

if [ "$mode" = check ]; then
  for key in calendar_url contacts_url admins_url sender emergency_email sendmail_path; do
    read_value "$key" > /dev/null
  done
  sendmail_path="$(read_value sendmail_path)"
  if [[ "$sendmail_path" != /* ]]; then
    echo "sendmail_path in $config_file muss ein absoluter Pfad sein." >&2
    exit 1
  fi
elif [ "$mode" != --cron ]; then
  echo "Unbekannter Modus: $mode" >&2
  exit 1
fi

cron_schedule="$(read_value cron_schedule)"
if ! [[ "$cron_schedule" =~ ^[0-9*,/-]+([[:space:]]+[0-9*,/-]+){4}$ ]]; then
  echo "Ungültiger cron_schedule in $config_file." >&2
  exit 1
fi
if [ "$mode" = --cron ]; then
  printf '%s\n' "$cron_schedule"
fi
