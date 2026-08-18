package plugin

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Manifest defines the metadata and security properties of a plugin
type Manifest struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	License     string   `json:"license"`
	Permissions []string `json:"permissions"` // e.g. ["network", "filesystem"]
	Signature   string   `json:"signature"`   // Hex-encoded Ed25519 signature of the content (excluding signature field)
}

// VerifySignature checks if the manifest signature is valid for the given public key
func (m *Manifest) VerifySignature(pubKeyHex string) error {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return fmt.Errorf("invalid public key hex: %w", err)
	}

	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(pubKey))
	}

	sig, err := hex.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %w", err)
	}

	// Canonicalize content for verification (exclude signature)
	// In a real system, we'd sign the hash of the plugin binary + manifest.
	// For now, we sign the JSON representation of the manifest fields.
	payload, err := m.CanonicalPayload()
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, payload, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// CanonicalPayload returns the byte slice to be signed
func (m *Manifest) CanonicalPayload() ([]byte, error) {
	// Create copy without signature
	clone := *m
	clone.Signature = ""
	return json.Marshal(clone)
}
