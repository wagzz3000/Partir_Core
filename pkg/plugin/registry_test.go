package plugin

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// MockPlugin for testing
type MockPlugin struct {
	id       string
	manifest *Manifest
}

func (m *MockPlugin) ID() string                                  { return "" } // Use manifest ID if available, or just empty for interface check
func (m *MockPlugin) Version() string                             { return "1.0.0" }
func (m *MockPlugin) Capabilities() []string                      { return []string{"test"} }
func (m *MockPlugin) InputSchema(jobType string) json.RawMessage  { return nil }
func (m *MockPlugin) OutputSchema(jobType string) json.RawMessage { return nil }
func (m *MockPlugin) Plan(workOrder WorkOrder) ([]Step, error)    { return nil, nil }
func (m *MockPlugin) Execute(ctx context.Context, workOrder WorkOrder) (*ExecutionResult, error) {
	return nil, nil
}
func (m *MockPlugin) Gates(jobType string) []string { return nil }
func (m *MockPlugin) Validate(ctx context.Context, artifacts []Artifact) ([]Defect, error) {
	return nil, nil
}
func (m *MockPlugin) Manifest() *Manifest { return m.manifest }

// Override ID for registration
func (m *MockPlugin) SetID(id string) { m.id = id }
func (m *MockPlugin) GetID() string   { return m.id }

// Fix interface implementation
func (m *MockPlugin) ID_Interface() string { return m.id }

// Re-implement ID() to match interface needed by Registry
// The interface method is ID() string.
// Go allows methods to be defined on pointer receiver.
// But we have conflicting method names if I add SetID/GetID helpers? No.
// Let's just use the strict interface.
// Wait, Register calls plugin.ID().
// So:
func (m *MockPlugin) RealID() string { return m.id }

// Shadowing?
// Lets redefine MockPlugin cleanly.

type TestPlugin struct {
	id       string
	manifest *Manifest
}

func (p *TestPlugin) ID() string                                  { return p.id }
func (p *TestPlugin) Version() string                             { return "1.0.0" }
func (p *TestPlugin) Manifest() *Manifest                         { return p.manifest }
func (p *TestPlugin) Capabilities() []string                      { return []string{"test"} }
func (p *TestPlugin) InputSchema(jobType string) json.RawMessage  { return nil }
func (p *TestPlugin) OutputSchema(jobType string) json.RawMessage { return nil }
func (p *TestPlugin) Plan(workOrder WorkOrder) ([]Step, error)    { return nil, nil }
func (p *TestPlugin) Execute(ctx context.Context, workOrder WorkOrder) (*ExecutionResult, error) {
	return nil, nil
}
func (p *TestPlugin) Gates(jobType string) []string { return nil }
func (p *TestPlugin) Validate(ctx context.Context, artifacts []Artifact) ([]Defect, error) {
	return nil, nil
}

func TestRegistry_Register_Signature(t *testing.T) {
	// 1. Generate Key Pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pubKeyHex := hex.EncodeToString(pubKey)

	// 2. Create Registry with Trust Root
	reg := NewRegistry()
	reg.SetTrustRoot(pubKeyHex)

	// 3. Create Plugin with Valid Signature
	manifest := &Manifest{
		ID:          "test-plugin",
		Version:     "1.0.0",
		Name:        "Test Plugin",
		Description: "A test plugin",
		Author:      "Test Author",
		License:     "MIT",
	}

	payload, _ := manifest.CanonicalPayload()
	sig := ed25519.Sign(privKey, payload)
	manifest.Signature = hex.EncodeToString(sig)

	plugin := &TestPlugin{
		id:       "test-plugin",
		manifest: manifest,
	}

	// 4. Register - Should Success
	if err := reg.Register(plugin); err != nil {
		t.Errorf("Register failed with valid signature: %v", err)
	}

	// 5. Create Plugin with Invalid Signature (Tampered)
	manifestBad := &Manifest{
		ID:          "bad-plugin",
		Version:     "1.0.0",
		Name:        "Bad Plugin",
		Description: "Tampered",
	}
	payloadBad, _ := manifestBad.CanonicalPayload()
	sigBad := ed25519.Sign(privKey, payloadBad) // valid signature for initial state
	manifestBad.Signature = hex.EncodeToString(sigBad)

	// Tamper
	manifestBad.Description = "Tampered Description"

	pluginBad := &TestPlugin{
		id:       "bad-plugin",
		manifest: manifestBad,
	}

	// 6. Register - Should Fail
	if err := reg.Register(pluginBad); err == nil {
		t.Error("Register should have failed with invalid signature")
	}
}

func TestRegistry_Register_NoTrustRoot(t *testing.T) {
	reg := NewRegistry() // No Trust Root

	plugin := &TestPlugin{
		id:       "unsigned-plugin",
		manifest: &Manifest{ID: "unsigned-plugin"}, // No signature
	}

	if err := reg.Register(plugin); err != nil {
		t.Errorf("Register failed without trust root: %v", err)
	}
}
