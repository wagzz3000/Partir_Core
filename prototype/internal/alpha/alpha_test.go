package alpha

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspace_Init(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alpha-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ws := NewWorkspace(tmpDir)
	name := "test-project"

	rb, err := ws.Init(name)
	assert.NoError(t, err)
	assert.NotNil(t, rb)
	assert.Equal(t, name, rb.Manifest.Name)

	// Verify dirs
	assert.DirExists(t, filepath.Join(tmpDir, name))
	assert.DirExists(t, filepath.Join(tmpDir, name, "schemas"))
	assert.DirExists(t, filepath.Join(tmpDir, name, "catalogs"))
	assert.DirExists(t, filepath.Join(tmpDir, name, "rules"))
	assert.FileExists(t, filepath.Join(tmpDir, name, "alpha_manifest.json"))
}

func TestWorkspace_Load(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "alpha-test-load")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ws := NewWorkspace(tmpDir)
	name := "project-load"
	_, err = ws.Init(name)
	assert.NoError(t, err)

	// Load it back
	loaded, err := ws.Load(name)
	assert.NoError(t, err)
	assert.Equal(t, name, loaded.Manifest.Name)
}

func TestLinter_Lint(t *testing.T) {
	linter := NewLinter()

	t.Run("Valid Rulebook", func(t *testing.T) {
		rb := NewRulebook("valid-rb")
		rb.Manifest.CoreVersion = "v0.1.0"
		rb.Compat.MinCoreVersion = "v0.1.0"

		result := linter.Lint(rb)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("Invalid Rulebook", func(t *testing.T) {
		rb := NewRulebook("") // Missing name
		// Missing CoreVersion

		result := linter.Lint(rb)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)

		// Check for specific errors
		foundNameErr := false
		for _, e := range result.Errors {
			if e.Path == "manifest.name" {
				foundNameErr = true
			}
		}
		assert.True(t, foundNameErr, "expected name error")
	})
}
