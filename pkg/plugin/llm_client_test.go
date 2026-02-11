package plugin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/partir/core/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMClient_Execute(t *testing.T) {
	// Mock server mimicking an LLM plugin
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/execute", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var workOrder plugin.HTTPWorkOrder
		err := json.NewDecoder(r.Body).Decode(&workOrder)
		require.NoError(t, err)

		// Verify input mapping
		assert.Contains(t, workOrder.Prompt, "Hello")

		// Response
		resp := plugin.RunResult{
			AgentResult: plugin.AgentResult{
				Success:   true,
				Artifacts: []plugin.AgentArtifact{},
			},
			Output: "World",
		}

		// Fill telemetry to avoid 0s if checked
		resp.Telemetry.FinishReason = "stop"

		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Initialize LLMClient with mock server URL
	httpClient := plugin.NewHTTPClient("mock-llm", ts.URL)
	client := plugin.NewLLMClientWithHTTP(httpClient)

	// Execute request
	req := &plugin.ExecuteRequest{
		TicketID: "t-123",
		RunID:    "r-456",
		Prompt:   "Hello",
	}

	resp, err := client.Execute(context.Background(), req)
	require.NoError(t, err)

	// Verify response mapping
	assert.Equal(t, "World", resp.Output)
	assert.Equal(t, "stop", resp.FinishReason)
}
