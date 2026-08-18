package remote

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input    string
		wantUser string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{"deploy@192.168.1.100", "deploy", "192.168.1.100", 22, false},
		{"root@example.com:2222", "root", "example.com", 2222, false},
		{"badformat", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			target, err := ParseTarget(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantUser, target.User)
			assert.Equal(t, tt.wantHost, target.Host)
			assert.Equal(t, tt.wantPort, target.Port)
		})
	}
}

func TestSSHCommand(t *testing.T) {
	target := &SSHTarget{User: "deploy", Host: "10.0.0.1", Port: 22}
	cmd := target.SSHCommand()
	assert.Equal(t, []string{"ssh", "-p", "22", "deploy@10.0.0.1"}, cmd)
}

func TestSCPCommand(t *testing.T) {
	target := &SSHTarget{User: "root", Host: "srv.example.com", Port: 2222}
	cmd := target.SCPCommand("/tmp/plugin", "/opt/partir/plugins/my-plugin")
	assert.Equal(t, []string{"scp", "-P", "2222", "/tmp/plugin", "root@srv.example.com:/opt/partir/plugins/my-plugin"}, cmd)
}

func TestWorkerInstallScript(t *testing.T) {
	script := WorkerInstallScript("llama3:8b")
	assert.Contains(t, script, "ollama pull llama3:8b")
	assert.Contains(t, script, "useradd")
}

func TestSystemdUnitGenerate(t *testing.T) {
	unit := DefaultWorkerUnit()
	content, err := unit.Generate()
	assert.NoError(t, err)
	assert.Contains(t, content, "[Unit]")
	assert.Contains(t, content, "Partir Core Worker")
	assert.Contains(t, content, "NoNewPrivileges=yes")
	assert.Contains(t, content, "ProtectSystem=strict")
}

func TestSystemdInstallScript(t *testing.T) {
	unit := DefaultOllamaUnit()
	script, err := unit.InstallScript()
	assert.NoError(t, err)
	assert.True(t, strings.Contains(script, "systemctl enable ollama"))
	assert.True(t, strings.Contains(script, "systemctl start ollama"))
}
