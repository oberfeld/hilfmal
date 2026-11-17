package publisher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oberfeld.be/hilfmal/pkg/models"
)

func TestPublishWritesEscapedStaticHTML(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.html")
	outputPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(templatePath, []byte("<p>{{name}} {{phone}}</p>"), 0600); err != nil {
		t.Fatal(err)
	}

	publisher, err := NewFilePublisher(outputPath, templatePath)
	if err != nil {
		t.Fatal(err)
	}
	contact := &models.Contact{Name: `<Ada>`, Phone: `123 & "456"`}
	if err := publisher.Publish(contact); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("published file permissions = %04o, want 0644", info.Mode().Perm())
	}
	page := string(content)
	for _, expected := range []string{"&lt;Ada&gt;", "123 &amp; &#34;456&#34;"} {
		if !strings.Contains(page, expected) {
			t.Errorf("page does not contain escaped value %q: %s", expected, page)
		}
	}
	if strings.Contains(page, "<?php") || strings.Contains(page, "private@example.test") {
		t.Fatalf("page contains PHP or private contact data: %s", page)
	}
}

func TestPublishNilContactWritesUnavailableMessage(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.html")
	outputPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(templatePath, []byte("<p>{{name}} {{phone}}</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	publisher, err := NewFilePublisher(outputPath, templatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(nil); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "nicht verfügbar") {
		t.Fatalf("page missing unavailable message: %s", content)
	}
}

func TestPublishFailsWhenWebDirectoryDoesNotExist(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.html")
	if err := os.WriteFile(templatePath, []byte("page"), 0600); err != nil {
		t.Fatal(err)
	}
	publisher, err := NewFilePublisher(filepath.Join(directory, "missing", "index.html"), templatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(nil); err == nil {
		t.Fatal("expected missing web directory to fail")
	}
}
