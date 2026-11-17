package main

import (
	"fmt"
	"os"
)

const (
	cloudURL = "https://nextcloud.test"
	mailURL  = "http://mailpit:8025"
	davUser  = "hilfmal"
	davPass  = "integration-test-only"
	sender   = "hilfmal@example.test"
)

func main() {
	if os.Getenv("HILFMAL_INTEGRATION_TEST") != "1" {
		fail("integration tests may only run in the isolated test environment")
	}
	if err := run(); err != nil {
		fail("%v", err)
	}
	fmt.Println("Integration tests passed: Nextcloud, email, static HTML, and error handling.")
}

func run() error {
	if err := waitForServices(); err != nil {
		return err
	}

	paths, err := setupNextcloud()
	if err != nil {
		return err
	}
	// today anchors the calendar fixtures; fixedTime is 06:00 on that service day,
	// passed to application runs to make their service date deterministic.
	today, fixedTime, err := integrationDate()
	if err != nil {
		return err
	}
	if err := seedCalendar(paths.calendar, today); err != nil {
		return err
	}

	pagePath, cleanup, err := newPublicPage()
	if err != nil {
		return err
	}
	// Remove the temporary public page when this run exits, including on errors.
	defer cleanup()

	if err := verifyInitialRun(paths, pagePath, today, fixedTime); err != nil {
		return err
	}
	if err := verifyContactRefresh(paths, pagePath, fixedTime); err != nil {
		return err
	}
	if err := verifyProductionEntryPoint(paths, pagePath); err != nil {
		return err
	}
	return verifyFailureNotifications(paths, pagePath, fixedTime)
}

func fail(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
