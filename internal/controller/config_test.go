package v1alpha1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogTemplates(t *testing.T) {
	tests := []struct {
		name      string
		logToDisk bool
	}{
		{name: "container streams", logToDisk: false},
		{name: "persistent files", logToDisk: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := logConfigData{LogToDisk: tt.logToDisk}
			supervisor := renderTemplateFile(t, "aim_supervisord.conf", data)
			kolla := renderTemplateFile(t, "kolla_config.json", data)
			initScript := renderTemplateFile(t, "init.sh", data)

			if !json.Valid([]byte(kolla)) {
				t.Fatalf("rendered kolla_config.json is invalid JSON:\n%s", kolla)
			}

			if tt.logToDisk {
				requireContains(t, supervisor, "--log-file=/var/log/aim/")
				requireContains(t, supervisor, "stdout_logfile=NONE")
				requireContains(t, kolla, "\"path\": \"/var/log/aim\"")
				requireContains(t, initScript, "init_done")
				return
			}

			requireContains(t, supervisor, "logfile = /dev/stdout")
			requireContains(t, supervisor, "stdout_logfile=/dev/fd/1")
			requireContains(t, supervisor, "stderr_logfile=/dev/fd/2")
			requireContains(t, initScript, "POD_ORDINAL=\"${POD_NAME##*-}\"")
			requireContains(t, initScript, "[ \"$POD_ORDINAL\" != \"0\" ]")

			for name, content := range map[string]string{
				"aim_supervisord.conf": supervisor,
				"kolla_config.json":    kolla,
				"init.sh":              initScript,
			} {
				if strings.Contains(content, "/var/log/aim") {
					t.Errorf("%s references /var/log/aim when persistence is disabled", name)
				}
			}
			if strings.Contains(supervisor, "--log-file") {
				t.Error("aim_supervisord.conf sets --log-file when persistence is disabled")
			}
		})
	}
}

func renderTemplateFile(t *testing.T, name string, data logConfigData) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "templates", name))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := executeTemplate(name, string(content), data)
	if err != nil {
		t.Fatal(err)
	}
	return rendered
}

func requireContains(t *testing.T, content, substring string) {
	t.Helper()
	if !strings.Contains(content, substring) {
		t.Errorf("expected content to include %q:\n%s", substring, content)
	}
}
