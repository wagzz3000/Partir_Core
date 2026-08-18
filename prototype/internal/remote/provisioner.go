package remote

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// SSHTarget represents a remote host
type SSHTarget struct {
	User string
	Host string
	Port int
}

// ParseTarget parses "user@host" or "user@host:port"
func ParseTarget(target string) (*SSHTarget, error) {
	t := &SSHTarget{Port: 22}

	parts := strings.SplitN(target, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid target format: expected user@host, got %q", target)
	}
	t.User = parts[0]

	hostPort := strings.SplitN(parts[1], ":", 2)
	t.Host = hostPort[0]
	if len(hostPort) == 2 {
		if _, err := fmt.Sscanf(hostPort[1], "%d", &t.Port); err != nil {
			return nil, fmt.Errorf("invalid port: %s", hostPort[1])
		}
	}

	return t, nil
}

// SSHCommand returns the base SSH command for this target
func (t *SSHTarget) SSHCommand() []string {
	return []string{
		"ssh", "-p", fmt.Sprintf("%d", t.Port),
		fmt.Sprintf("%s@%s", t.User, t.Host),
	}
}

// SCPCommand returns the SCP command to copy a file to the target
func (t *SSHTarget) SCPCommand(localPath, remotePath string) []string {
	return []string{
		"scp", "-P", fmt.Sprintf("%d", t.Port),
		localPath,
		fmt.Sprintf("%s@%s:%s", t.User, t.Host, remotePath),
	}
}

// WorkerInstallScript generates the shell script for remote worker installation
func WorkerInstallScript(ollamaModel string) string {
	script := `#!/bin/bash
set -euo pipefail

echo "🔧 Installing Partir Worker + Ollama..."

# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull default model
echo "📦 Pulling model: {{.Model}}"
ollama pull {{.Model}}

# Create partir user
if ! id "partir" &>/dev/null; then
    sudo useradd -r -s /bin/false partir
fi

# Create directories
sudo mkdir -p /opt/partir/plugins
sudo chown -R partir:partir /opt/partir

echo "✅ Worker installation complete"
echo "   Ollama model: {{.Model}}"
echo "   Plugin dir:   /opt/partir/plugins"
`
	tmpl := template.Must(template.New("worker").Parse(script))
	var buf bytes.Buffer
	tmpl.Execute(&buf, map[string]string{"Model": ollamaModel})
	return buf.String()
}

// PluginInstallScript generates the shell script to install a plugin on a remote node
func PluginInstallScript(slug, downloadURL string) string {
	script := `#!/bin/bash
set -euo pipefail

echo "📦 Installing plugin: {{.Slug}}"

PLUGIN_DIR="/opt/partir/plugins"
mkdir -p "$PLUGIN_DIR"

# Download plugin
curl -fsSL -o "$PLUGIN_DIR/{{.Slug}}" "{{.URL}}"
chmod +x "$PLUGIN_DIR/{{.Slug}}"

echo "✅ Plugin {{.Slug}} installed to $PLUGIN_DIR/{{.Slug}}"
`
	tmpl := template.Must(template.New("plugin").Parse(script))
	var buf bytes.Buffer
	tmpl.Execute(&buf, map[string]string{"Slug": slug, "URL": downloadURL})
	return buf.String()
}
