package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a persistent Round Table session
type Session struct {
	ID          string                 `json:"id"`
	TicketID    string                 `json:"ticket_id"`
	State       string                 `json:"state"`        // "active", "paused", "completed"
	CurrentStep string                 `json:"current_step"` // Where we left off
	Attempt     int                    `json:"attempt"`
	Context     map[string]interface{} `json:"context,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

const sessionDir = ".partir"
const sessionFile = "session.json"

// Save persists the session to disk
func (s *Session) Save() error {
	dir := sessionDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session dir: %w", err)
	}

	s.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := filepath.Join(dir, sessionFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// Load reads the session from disk
func Load() (*Session, error) {
	path := filepath.Join(sessionDir, sessionFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No session
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &s, nil
}

// Clear removes the session file
func Clear() error {
	path := filepath.Join(sessionDir, sessionFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	return nil
}

// NewSession creates a fresh session
func NewSession(ticketID string) *Session {
	return &Session{
		ID:        fmt.Sprintf("ses_%d", time.Now().UnixNano()),
		TicketID:  ticketID,
		State:     "active",
		Attempt:   1,
		Context:   make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
