package publisher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"oberfeld.be/hilfmal/pkg/models"
)

// @lat: [[tests#Tests#Publication and mail safety#Escaped static page]]
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
	if !utf8.Valid(content) {
		t.Fatal("unavailable page is not valid UTF-8")
	}
	page := string(content)
	for _, expected := range []string{
		"<!doctype html>",
		`<meta charset="utf-8">`,
		"Eine verlässliche Dienstinformation ist zurzeit nicht verfügbar.",
	} {
		if !strings.Contains(page, expected) {
			t.Errorf("unavailable page missing %q: %s", expected, page)
		}
	}
}

func TestWebTemplateShowsCallLinkOrUnavailableMessage(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "..", "templates", "web.html"))
	if err != nil {
		t.Fatal(err)
	}
	publisher := &FilePublisher{template: string(template)}

	page := publisher.render(&models.Contact{Name: "Léa", Phone: "+41 79 123 45 67"})
	for _, expected := range []string{
		`<section  aria-label="Diensthabende Person">`,
		`<section hidden aria-label="Keine Dienstinformation">`,
		`href="tel:+41791234567"`,
		"+41 79 123 45 67",
		"Léa",
	} {
		if !strings.Contains(page, expected) {
			t.Errorf("contact page missing %q", expected)
		}
	}

	page = publisher.render(nil)
	for _, expected := range []string{
		`<section hidden aria-label="Diensthabende Person">`,
		`<section  aria-label="Keine Dienstinformation">`,
		`<meta charset="utf-8">`,
		"Eine verlässliche Dienstinformation ist zurzeit nicht verfügbar.",
	} {
		if !strings.Contains(page, expected) {
			t.Errorf("unavailable page missing %q", expected)
		}
	}
}

func TestCallLinkEscapesPhoneAndReplacesPlaceholdersOnce(t *testing.T) {
	publisher := &FilePublisher{template: `<a href="{{phone_url}}">{{name}} {{phone}}</a>`}
	page := publisher.render(&models.Contact{Name: `{{phone}}`, Phone: `tel:+41 79 "12"&3`})
	if !strings.Contains(page, `href="tel:+4179%2212%22&amp;3"`) {
		t.Errorf("phone URL was not safely escaped: %s", page)
	}
	if !strings.Contains(page, `>{{phone}} tel:+41 79 &#34;12&#34;&amp;3</a>`) {
		t.Errorf("contact fields were not safely rendered: %s", page)
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
