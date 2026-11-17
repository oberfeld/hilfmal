package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Second}

func waitForServices() error {
	if err := waitForService(checkNextcloudReady, "Nextcloud"); err != nil {
		return err
	}
	return waitForService(checkMailpitReady, "Mailpit")
}

func checkNextcloudReady() error {
	status, body, err := sendHTTPRequest(http.MethodGet, cloudURL+"/status.php", nil)
	if err != nil {
		return err
	}
	var data struct {
		Installed bool `json:"installed"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}
	if status != http.StatusOK || !data.Installed {
		return fmt.Errorf("Nextcloud is not installed yet")
	}
	return nil
}

func checkMailpitReady() error {
	status, _, err := sendHTTPRequest(http.MethodGet, mailURL+"/api/v1/info", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("Mailpit returned HTTP %d", status)
	}
	return nil
}

func sendHTTPRequest(method, address string, body []byte) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, address, reader)
	if err != nil {
		return 0, nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	return response.StatusCode, content, err
}

func sendDAVRequest(method, path, body, contentType string, expected ...int) error {
	req, err := http.NewRequest(method, cloudURL+path, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(davUser, davPass)
	req.Header.Set("Content-Type", contentType)
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	for _, status := range expected {
		if response.StatusCode == status {
			return nil
		}
	}
	return fmt.Errorf("%s %s failed (HTTP %d)", method, path, response.StatusCode)
}

// waitForService synchronously polls ready every 500ms until it succeeds or times out.
func waitForService(ready func() error, name string) error {
	deadline := time.Now().Add(180 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := ready(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s was not ready after 180 seconds: %w", name, lastErr)
}

func clearMailpitMessages() error {
	status, _, err := sendHTTPRequest(http.MethodDelete, mailURL+"/api/v1/messages", nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("Mailpit could not be cleared (HTTP %d)", status)
	}
	return nil
}

func verifyMailpitMessage(recipient, content string) error {
	status, body, err := sendHTTPRequest(http.MethodGet, mailURL+"/api/v1/messages", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("Mailpit messages returned HTTP %d", status)
	}
	var result struct {
		Messages []struct {
			ID string `json:"ID"`
			To []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	for _, message := range result.Messages {
		for _, to := range message.To {
			if to.Address != recipient {
				continue
			}
			address := mailURL + "/api/v1/message/" + url.PathEscape(message.ID)
			status, body, err := sendHTTPRequest(http.MethodGet, address, nil)
			if err != nil {
				return err
			}
			if status != http.StatusOK {
				return fmt.Errorf("Mailpit message returned HTTP %d", status)
			}
			var detail struct {
				Text string `json:"Text"`
				From struct {
					Address string `json:"Address"`
				} `json:"From"`
			}
			if err := json.Unmarshal(body, &detail); err != nil {
				return err
			}
			if detail.From.Address == sender && strings.Contains(detail.Text, content) {
				return nil
			}
		}
	}
	return fmt.Errorf("expected email to %s containing %q was not found", recipient, content)
}
