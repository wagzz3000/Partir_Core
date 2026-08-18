package remote

import (
	"bytes"
	"fmt"
	"text/template"
)

// SystemdUnit represents a systemd service unit configuration
type SystemdUnit struct {
	Name        string
	Description string
	ExecStart   string
	User        string
	Group       string
	WorkingDir  string
	Environment map[string]string
	After       []string
	Restart     string
	RestartSec  int
}

// DefaultWorkerUnit returns a systemd unit for the Partir worker
func DefaultWorkerUnit() *SystemdUnit {
	return &SystemdUnit{
		Name:        "partir-worker",
		Description: "Partir Core Worker",
		ExecStart:   "/usr/local/bin/foundry serve",
		User:        "partir",
		Group:       "partir",
		WorkingDir:  "/opt/partir",
		Environment: map[string]string{
			"PARTIR_DB_URL":         "",
			"NATS_URL":              "",
			"PARTIR_MINIO_ENDPOINT": "",
			"PARTIR_METRICS_PORT":   "9090",
		},
		After:      []string{"network.target", "postgresql.service"},
		Restart:    "always",
		RestartSec: 5,
	}
}

// DefaultOllamaUnit returns a systemd unit for Ollama
func DefaultOllamaUnit() *SystemdUnit {
	return &SystemdUnit{
		Name:        "ollama",
		Description: "Ollama LLM Server",
		ExecStart:   "/usr/local/bin/ollama serve",
		User:        "partir",
		Group:       "partir",
		WorkingDir:  "/opt/partir",
		Environment: map[string]string{
			"OLLAMA_HOST": "0.0.0.0:11434",
		},
		After:      []string{"network.target"},
		Restart:    "always",
		RestartSec: 3,
	}
}

// Generate produces the systemd unit file content
func (u *SystemdUnit) Generate() (string, error) {
	const unitTemplate = `[Unit]
Description={{.Description}}
{{- range .After}}
After={{.}}
{{- end}}

[Service]
Type=simple
User={{.User}}
Group={{.Group}}
WorkingDirectory={{.WorkingDir}}
ExecStart={{.ExecStart}}
Restart={{.Restart}}
RestartSec={{.RestartSec}}
{{range $k, $v := .Environment -}}
Environment="{{$k}}={{$v}}"
{{end}}
# Hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/partir

[Install]
WantedBy=multi-user.target
`
	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, u); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// InstallScript returns a shell script that writes and enables the unit
func (u *SystemdUnit) InstallScript() (string, error) {
	unitContent, err := u.Generate()
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "📦 Installing systemd unit: %s"

cat > /etc/systemd/system/%s.service << 'UNIT_EOF'
%sUNIT_EOF

systemctl daemon-reload
systemctl enable %s
systemctl start %s

echo "✅ Service %s started"
systemctl status %s --no-pager
`, u.Name, u.Name, unitContent, u.Name, u.Name, u.Name, u.Name)

	return script, nil
}
