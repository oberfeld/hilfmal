package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// @lat: [[lat.md/tests#Tests#DAV transport and integration#Safe cron updates]]
func TestUpdateCrontabPreservesOtherJobsAndReplacesHilfmalJob(t *testing.T) {
	state := filepath.Join(t.TempDir(), "crontab")
	const oldCrontab = "0 4 * * * /usr/local/bin/backup\n0 6 * * * old-hilfmal # hilfmal deployment\n"
	if err := os.WriteFile(state, []byte(oldCrontab), 0600); err != nil {
		t.Fatal(err)
	}

	const cronJob = "0 6 * * * /home/user/hilfmal # hilfmal deployment"
	runCrontabUpdate(t, state, cronJob, "")
	runCrontabUpdate(t, state, cronJob, "")

	updated, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(updated), "0 4 * * * /usr/local/bin/backup\n"+cronJob+"\n"; got != want {
		t.Fatalf("updated crontab = %q, want %q", got, want)
	}
}

func TestUpdateCrontabCreatesFirstCrontab(t *testing.T) {
	state := filepath.Join(t.TempDir(), "crontab")
	const cronJob = "0 6 * * * /home/user/hilfmal # hilfmal deployment"

	runCrontabUpdate(t, state, cronJob, "")

	updated, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(updated); got != cronJob+"\n" {
		t.Fatalf("updated crontab = %q, want %q", got, cronJob+"\n")
	}
}

func TestUpdateCrontabDoesNotWriteAfterReadFailure(t *testing.T) {
	state := filepath.Join(t.TempDir(), "crontab")
	const original = "0 4 * * * /usr/local/bin/backup\n"
	if err := os.WriteFile(state, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := runCrontabUpdateCommand(t, state, "0 6 * * * /home/user/hilfmal # hilfmal deployment", "1")
	if err == nil || !strings.Contains(string(output), "permission denied") {
		t.Fatalf("update output = %q, error = %v; want read failure", output, err)
	}

	updated, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(updated); got != original {
		t.Fatalf("crontab after read failure = %q, want unchanged %q", got, original)
	}
}

func runCrontabUpdate(t *testing.T, state, cronJob, readError string) {
	t.Helper()
	output, err := runCrontabUpdateCommand(t, state, cronJob, readError)
	if err != nil {
		t.Fatalf("crontab update failed: %v: %s", err, output)
	}
}

func runCrontabUpdateCommand(t *testing.T, state, cronJob, readError string) ([]byte, error) {
	t.Helper()
	fakeCrontab := filepath.Join(t.TempDir(), "fake-crontab")
	const fakeScript = `#!/bin/sh
set -eu
case "$1" in
  -l)
    if [ "${FAKE_CRONTAB_READ_ERROR:-}" = "1" ]; then
      echo "permission denied" >&2
      exit 2
    fi
    if [ -f "$FAKE_CRONTAB_STATE" ]; then
      cat "$FAKE_CRONTAB_STATE"
    else
      echo "no crontab for test-user" >&2
      exit 1
    fi
    ;;
  -)
    cat > "$FAKE_CRONTAB_STATE"
    ;;
  *)
    exit 2
    ;;
esac
`
	if err := os.WriteFile(fakeCrontab, []byte(fakeScript), 0700); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("sh", filepath.Join("..", "scripts", "update-crontab.sh"), cronJob)
	command.Env = append(os.Environ(),
		"CRONTAB_COMMAND="+fakeCrontab,
		"FAKE_CRONTAB_STATE="+state,
		"FAKE_CRONTAB_READ_ERROR="+readError,
	)
	return command.CombinedOutput()
}
