package publisher

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"oberfeld.be/hilfmal/pkg/models"
)

type Publisher interface {
	Publish(contact *models.Contact) error
}

type FilePublisher struct {
	path     string
	template string
}

func NewFilePublisher(path, templatePath string) (*FilePublisher, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read web template: %w", err)
	}
	return &FilePublisher{path: path, template: string(content)}, nil
}

func (p *FilePublisher) Publish(contact *models.Contact) error {
	content := p.render(contact)
	directory := filepath.Dir(p.path)
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("web directory is unavailable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("web path parent is not a directory")
	}

	file, err := os.CreateTemp(directory, ".hilfmal-*")
	if err != nil {
		return fmt.Errorf("create temporary web file: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)

	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary web file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary web file: %w", err)
	}
	if err := file.Chmod(0644); err != nil {
		_ = file.Close()
		return fmt.Errorf("set web file permissions: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary web file: %w", err)
	}
	if err := os.Rename(temporary, p.path); err != nil {
		return fmt.Errorf("publish web page: %w", err)
	}
	return nil
}

func (p *FilePublisher) render(contact *models.Contact) string {
	if contact == nil && (!strings.Contains(p.template, "{{contact_hidden}}") || !strings.Contains(p.template, "{{unavailable_hidden}}")) {
		return `<!doctype html>
<html lang="de">
<head><meta charset="utf-8"></head>
<body><p>Eine verlässliche Dienstinformation ist zurzeit nicht verfügbar.</p></body>
</html>
`
	}
	name, phone, phoneURL := "", "", ""
	contactHidden, unavailableHidden := "hidden", ""
	if contact != nil {
		name = html.EscapeString(contact.Name)
		phone = html.EscapeString(contact.Phone)
		number := strings.Join(strings.Fields(contact.Phone), "")
		if strings.HasPrefix(strings.ToLower(number), "tel:") {
			number = number[len("tel:"):]
		}
		phoneURL = html.EscapeString("tel:" + url.PathEscape(number))
		contactHidden, unavailableHidden = "", "hidden"
	}
	return strings.NewReplacer(
		"{{name}}", name,
		"{{phone}}", phone,
		"{{phone_url}}", phoneURL,
		"{{contact_hidden}}", contactHidden,
		"{{unavailable_hidden}}", unavailableHidden,
	).Replace(p.template)
}
