package chatops

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Command represents a parsed chat command
type Command struct {
	Name    string   // "status", "approve", "help"
	Args    []string // Additional arguments
	UserID  string   // Who sent it
	Channel string   // Where it was sent
}

// Handler processes chat commands and returns responses
type Handler interface {
	HandleCommand(ctx context.Context, cmd Command) (*Response, error)
}

// Response is sent back to the chat platform
type Response struct {
	Text   string  `json:"text"`
	Blocks []Block `json:"blocks,omitempty"` // Slack blocks / Discord embeds
}

// Block represents a rich message block
type Block struct {
	Type string `json:"type"` // "section", "divider", "context"
	Text string `json:"text,omitempty"`
}

// --- Slack Bot ---

// SlackBot handles Slack slash commands and events
type SlackBot struct {
	handler       Handler
	signingSecret string
}

// NewSlackBot creates a new Slack bot
func NewSlackBot(handler Handler, signingSecret string) *SlackBot {
	return &SlackBot{
		handler:       handler,
		signingSecret: signingSecret,
	}
}

// SlashCommandHandler handles Slack slash commands (/partir status, /partir approve)
func (sb *SlackBot) SlashCommandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		text := r.FormValue("text")
		userID := r.FormValue("user_id")
		channelID := r.FormValue("channel_id")

		cmd := parseCommand(text, userID, channelID)

		resp, err := sb.handler.HandleCommand(r.Context(), cmd)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"response_type": "ephemeral",
				"text":          fmt.Sprintf("❌ Error: %s", err.Error()),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response_type": "in_channel",
			"text":          resp.Text,
		})
	}
}

// --- Discord Bot ---

// DiscordBot handles Discord interactions with Ed25519 signature verification
type DiscordBot struct {
	handler   Handler
	token     string
	publicKey ed25519.PublicKey
}

// NewDiscordBot creates a new Discord bot with Ed25519 public key for verification
func NewDiscordBot(handler Handler, token, publicKeyHex string) (*DiscordBot, error) {
	keyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key hex: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}

	return &DiscordBot{
		handler:   handler,
		token:     token,
		publicKey: ed25519.PublicKey(keyBytes),
	}, nil
}

// InteractionHandler handles Discord slash command interactions with signature verification
func (db *DiscordBot) InteractionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read body for signature verification
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify Ed25519 signature (required by Discord)
		signature := r.Header.Get("X-Signature-Ed25519")
		timestamp := r.Header.Get("X-Signature-Timestamp")

		if !db.verifySignature(signature, timestamp, body) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		// Parse the interaction
		var interaction struct {
			Type int `json:"type"`
			Data struct {
				Name    string `json:"name"`
				Options []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"options"`
			} `json:"data"`
			Member struct {
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"member"`
			ChannelID string `json:"channel_id"`
		}

		if err := json.Unmarshal(body, &interaction); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		// Type 1 = PING — Discord requires an immediate PONG
		if interaction.Type == 1 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{"type": 1})
			return
		}

		// Type 2 = APPLICATION_COMMAND
		if interaction.Type == 2 {
			// Build args from options
			var args []string
			for _, opt := range interaction.Data.Options {
				args = append(args, opt.Value)
			}

			cmd := Command{
				Name:    interaction.Data.Name,
				Args:    args,
				UserID:  interaction.Member.User.ID,
				Channel: interaction.ChannelID,
			}

			resp, err := db.handler.HandleCommand(r.Context(), cmd)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"type": 4,
					"data": map[string]string{"content": fmt.Sprintf("❌ %s", err.Error())},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"type": 4,
				"data": map[string]string{"content": resp.Text},
			})
			return
		}

		http.Error(w, "unknown interaction type", http.StatusBadRequest)
	}
}

// verifySignature checks the Discord Ed25519 signature
func (db *DiscordBot) verifySignature(signatureHex, timestamp string, body []byte) bool {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	msg := append([]byte(timestamp), body...)
	return ed25519.Verify(db.publicKey, msg, sig)
}

// --- Status Provider ---

// StatusProvider supplies live system status for the ChatOps status command
type StatusProvider interface {
	GetQueueDepth(ctx context.Context) (int, error)
	GetActiveTickets(ctx context.Context) (int, error)
	GetLastCompletedAt(ctx context.Context) (*time.Time, error)
}

// DefaultHandler implements core chat commands with live status
type DefaultHandler struct {
	status StatusProvider // nil = static placeholder
}

// NewDefaultHandler creates a handler; pass nil for static responses
func NewDefaultHandler(status StatusProvider) *DefaultHandler {
	return &DefaultHandler{status: status}
}

func (h *DefaultHandler) HandleCommand(ctx context.Context, cmd Command) (*Response, error) {
	switch cmd.Name {
	case "status":
		return h.statusResponse(ctx)
	case "approve":
		if len(cmd.Args) == 0 {
			return &Response{Text: "Usage: `/partir approve <ticket-id>`"}, nil
		}
		return &Response{
			Text: fmt.Sprintf("✅ Ticket `%s` approved by <@%s>", cmd.Args[0], cmd.UserID),
		}, nil
	case "help":
		return &Response{
			Text: "🔧 *Partir Commands*\n• `/partir status` — View factory status\n• `/partir approve <id>` — Approve a ticket\n• `/partir help` — Show this help",
		}, nil
	default:
		return &Response{
			Text: fmt.Sprintf("Unknown command: `%s`. Try `/partir help`.", cmd.Name),
		}, nil
	}
}

func (h *DefaultHandler) statusResponse(ctx context.Context) (*Response, error) {
	if h.status == nil {
		return &Response{
			Text: "📊 *Partir Status*\n• StatusProvider not configured. Connect a StatusProvider for live data.",
		}, nil
	}

	queueDepth, _ := h.status.GetQueueDepth(ctx)
	activeTickets, _ := h.status.GetActiveTickets(ctx)
	lastCompleted, _ := h.status.GetLastCompletedAt(ctx)

	lastStr := "N/A"
	if lastCompleted != nil {
		lastStr = lastCompleted.Format(time.RFC3339)
	}

	return &Response{
		Text: fmt.Sprintf("📊 *Partir Status*\n• Active tickets: %d\n• Queue depth: %d\n• Last completed: %s",
			activeTickets, queueDepth, lastStr),
	}, nil
}

func parseCommand(text, userID, channel string) Command {
	parts := strings.Fields(strings.TrimSpace(text))
	cmd := Command{
		UserID:  userID,
		Channel: channel,
		Name:    "help",
	}
	if len(parts) > 0 {
		cmd.Name = strings.ToLower(parts[0])
	}
	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}
	return cmd
}
