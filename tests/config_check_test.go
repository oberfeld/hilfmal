package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCyonConfigCheck(t *testing.T) {
	const valid = `calendar_url: "https://example.test/calendar"
contacts_url: "https://example.test/contacts"
admins_url: "https://example.test/admins"
sender: "sender@example.test"
emergency_email: "admin@example.test"
cron_schedule: "0 6 * * *"
`
	tests := []struct {
		name    string
		config  string
		mode    string
		want    string
		wantErr string
	}{
		{name: "valid", config: valid},
		{name: "read cron", config: valid, mode: "--cron", want: "0 6 * * *\n"},
		{name: "missing field", config: strings.Replace(valid, "sender: \"sender@example.test\"\n", "", 1), wantErr: "sender"},
		{name: "empty field", config: strings.Replace(valid, `"sender@example.test"`, `""`, 1), wantErr: "sender"},
		{name: "duplicate field", config: valid + "sender: \"other@example.test\"\n", wantErr: "sender"},
		{name: "invalid cron", config: strings.Replace(valid, "0 6 * * *", "0 6 * * *; id", 1), mode: "--cron", wantErr: "cron_schedule"},
		{name: "missing cron", config: strings.Replace(valid, "cron_schedule: \"0 6 * * *\"\n", "", 1), mode: "--cron", wantErr: "cron_schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.config), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"../scripts/check-cyon-config.sh", path}
			if tt.mode != "" {
				args = append(args, tt.mode)
			}
			output, err := exec.Command("bash", args...).CombinedOutput()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(string(output), tt.wantErr) {
					t.Fatalf("check = %q, %v; want error containing %q", output, err, tt.wantErr)
				}
				return
			}
			if err != nil || string(output) != tt.want {
				t.Fatalf("check = %q, %v; want %q, no error", output, err, tt.want)
			}
		})
	}
}
