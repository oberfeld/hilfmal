package mailer

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// @lat: [[tests#Tests#Publication and mail safety#MIME and local submission]]
func TestBuildMessageEncodesSubjectAndUsesUTF8(t *testing.T) {
	message, recipient, sender, err := buildMessage(
		"person@example.test",
		"sender@example.test",
		"Erinnerung an deinen Hilfmal-Dienst",
		"Grüezi\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	if recipient.Address != "person@example.test" || sender.Address != "sender@example.test" {
		t.Fatalf("unexpected envelope addresses: %q, %q", recipient.Address, sender.Address)
	}
	content := string(message)
	if !strings.Contains(content, "Content-Type: text/plain; charset=UTF-8\r\n") ||
		!strings.Contains(content, "Grüezi\n") {
		t.Fatalf("message missing UTF-8 headers or body: %q", content)
	}
	subjectLine := strings.SplitN(content, "\r\n", 4)[3]
	subject := strings.TrimPrefix(strings.SplitN(subjectLine, "\r\n", 2)[0], "Subject: ")
	decoded, err := new(mime.WordDecoder).DecodeHeader(subject)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "Erinnerung an deinen Hilfmal-Dienst" {
		t.Fatalf("decoded subject = %q", decoded)
	}
}

func TestBuildMessageRejectsHeaderInjection(t *testing.T) {
	if _, _, _, err := buildMessage("person@example.test\r\nBcc: other@example.test", "sender@example.test", "hello", "body"); err == nil {
		t.Fatal("expected recipient header injection to be rejected")
	}
	if _, _, _, err := buildMessage("person@example.test", "sender@example.test", "hello\r\nBcc: other@example.test", "body"); err == nil {
		t.Fatal("expected subject header injection to be rejected")
	}
}

func TestSystemMailerReturnsCommandFailure(t *testing.T) {
	mailer := NewSystemMailer("/path/that/does/not/exist/sendmail")
	if err := mailer.Send("person@example.test", "sender@example.test", "subject", "body"); err == nil {
		t.Fatal("expected missing local mail command to fail")
	}
}

func TestSystemMailerSubmitsMessageToConfiguredCommand(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "sendmail")
	capture := filepath.Join(directory, "message.eml")
	script := "#!/bin/sh\n[ \"$1\" = \"-t\" ] && [ \"$2\" = \"-i\" ] || exit 2\ncat > \"$HILFMAL_CAPTURE\"\n"
	if err := os.WriteFile(command, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HILFMAL_CAPTURE", capture)

	mailer := NewSystemMailer(command)
	if err := mailer.Send("person@example.test", "sender@example.test", "subject", "body"); err != nil {
		t.Fatal(err)
	}
	message, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(message), "To: <person@example.test>\r\n") ||
		!strings.Contains(string(message), "\r\n\r\nbody") {
		t.Fatalf("unexpected submitted message: %q", message)
	}
}
