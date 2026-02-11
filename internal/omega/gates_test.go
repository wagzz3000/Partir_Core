package omega

import (
	"context"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
)

func TestSchemaGate(t *testing.T) {
	gate := NewSchemaGate()
	schemaStr := `{
		"type": "object",
		"properties": {
			"foo": {"type": "string"}
		},
		"required": ["foo"]
	}`
	schema, err := jsonschema.CompileString("test.json", schemaStr)
	assert.NoError(t, err)
	gate.RegisterSchema("test-schema", schema)

	t.Run("Valid Artifact", func(t *testing.T) {
		validJSON := []byte(`{"foo": "bar"}`)
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a1", SchemaRef: "test-schema", Data: validJSON},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.Empty(t, defects)
	})

	t.Run("Invalid Artifact", func(t *testing.T) {
		invalidJSON := []byte(`{"foo": 123}`) // foo should be string
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a2", SchemaRef: "test-schema", Data: invalidJSON},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Equal(t, DefectClassSchema, defects[0].DefectClass)
		assert.Contains(t, defects[0].Message, "expected string")
	})

	t.Run("Missing Schema", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a3", SchemaRef: "unknown-schema", Data: []byte("{}")},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Contains(t, defects[0].Message, "schema not found")
	})
}

func TestVersionCompatGate(t *testing.T) {
	gate := NewVersionCompatGate("v0.1.0")

	t.Run("Matching Version", func(t *testing.T) {
		req := &GateRequest{CoreVersion: "v0.1.0"}
		defects := gate.Run(context.Background(), req)
		assert.Empty(t, defects)
	})

	t.Run("Mismatch Version", func(t *testing.T) {
		req := &GateRequest{CoreVersion: "v0.2.0"}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Equal(t, DefectClassVersion, defects[0].DefectClass)
		assert.Contains(t, defects[0].Message, "expected v0.1.0, got v0.2.0")
	})
}

func TestDeterminismGate(t *testing.T) {
	gate := NewDeterminismGate()

	t.Run("With Hash", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a1", Hash: "sha256:1234"},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.Empty(t, defects)
	})

	t.Run("Missing Hash", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a2", Hash: ""},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Equal(t, DefectClassDeterminism, defects[0].DefectClass)
		assert.Contains(t, defects[0].Message, "missing hash")
	})
}

func TestUniquenessGate(t *testing.T) {
	mockExists := func(ctx context.Context, hash string) (bool, error) {
		if hash == "exists" {
			return true, nil
		}
		return false, nil
	}
	gate := NewUniquenessGate(mockExists)

	t.Run("Unique Artifacts", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a1", Hash: "new1"},
				{ArtifactID: "a2", Hash: "new2"},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.Empty(t, defects)
	})

	t.Run("Batch Duplicate", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a1", Hash: "dup"},
				{ArtifactID: "a2", Hash: "dup"},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Equal(t, DefectClassUniqueness, defects[0].DefectClass)
		assert.Contains(t, defects[0].Message, "duplicate artifact in batch")
	})

	t.Run("Global Duplicate", func(t *testing.T) {
		req := &GateRequest{
			Artifacts: []ArtifactData{
				{ArtifactID: "a1", Hash: "exists"},
			},
		}
		defects := gate.Run(context.Background(), req)
		assert.NotEmpty(t, defects)
		assert.Contains(t, defects[0].Message, "artifact already exists")
	})
}
