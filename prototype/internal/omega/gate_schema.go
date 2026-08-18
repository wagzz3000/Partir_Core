package omega

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaGate validates artifacts against JSON schemas
type SchemaGate struct {
	schemas map[string]*jsonschema.Schema
}

// NewSchemaGate creates a new schema validation gate
func NewSchemaGate() *SchemaGate {
	return &SchemaGate{
		schemas: make(map[string]*jsonschema.Schema),
	}
}

func (g *SchemaGate) ID() string   { return "schema" }
func (g *SchemaGate) Name() string { return "Schema Validation" }

// RegisterSchema adds a schema for validation
func (g *SchemaGate) RegisterSchema(name string, schema *jsonschema.Schema) {
	g.schemas[name] = schema
}

func (g *SchemaGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	for _, artifact := range req.Artifacts {
		if artifact.SchemaRef == "" {
			continue
		}

		schema, ok := g.schemas[artifact.SchemaRef]
		if !ok {
			defects = append(defects, *NewDefect(g.ID(), DefectClassSchema,
				fmt.Sprintf("schema not found: %s", artifact.SchemaRef)))
			continue
		}

		var data interface{}
		if err := json.Unmarshal(artifact.Data, &data); err != nil {
			defects = append(defects, *NewDefect(g.ID(), DefectClassSchema,
				fmt.Sprintf("invalid JSON: %v", err)))
			continue
		}

		if err := schema.Validate(data); err != nil {
			msg := err.Error()
			fix := "Review the schema definition and adjust the JSON input."

			// Simple heuristic for common errors
			if _, ok := err.(*jsonschema.ValidationError); ok {
				// We could parse err.Message/InstancePtr but jsonschema error strings are complex
				// "I[#] S[#/properties/foo/type] expected string, but got number"
				// Let's rely on the message for now
			}

			d := NewDefect(g.ID(), DefectClassSchema, msg)
			d.WithOffendingFields(map[string]string{"artifact_id": artifact.ArtifactID})
			d.SuggestedFix = fix
			defects = append(defects, *d)
		}
	}

	return defects
}
