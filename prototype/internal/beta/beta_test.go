package beta

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspace_Init(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "beta-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ws := NewWorkspace(tmpDir)
	name := "test-style"

	rb, err := ws.Init(name)
	assert.NoError(t, err)
	assert.NotNil(t, rb)
	assert.Equal(t, name, rb.Manifest.Name)

	// Verify files
	assert.FileExists(t, filepath.Join(tmpDir, name, "beta_manifest.json"))
	assert.FileExists(t, filepath.Join(tmpDir, name, "render_vocab.json"))
	assert.FileExists(t, filepath.Join(tmpDir, name, "style_rules.json"))

	// Verify defaults
	assert.NotEmpty(t, rb.RenderVocab.Effects)
	assert.NotEmpty(t, rb.RenderVocab.Palettes)
}

func TestWorkspace_Load(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "beta-test-load")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ws := NewWorkspace(tmpDir)
	name := "style-load"
	_, err = ws.Init(name)
	assert.NoError(t, err)

	// Load it back
	loaded, err := ws.Load(name)
	assert.NoError(t, err)
	assert.Equal(t, name, loaded.Manifest.Name)
	assert.NotEmpty(t, loaded.RenderVocab.Effects)
}
