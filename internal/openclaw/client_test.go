package openclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// mockGateway runs a mock WebSocket server that replays the challenge/response sequence.
func mockGateway(t *testing.T, password string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Step 1: Send connect.challenge
		challenge := EventFrame{
			Type:  FrameTypeEvent,
			Event: "connect.challenge",
			Payload: mustMarshal(ChallengePayload{
				Nonce: "test-nonce-12345",
			}),
		}
		if err := conn.WriteJSON(challenge); err != nil {
			return
		}

		// Step 2: Read connect request
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var req RequestFrame
		if err := json.Unmarshal(msg, &req); err != nil {
			return
		}

		if req.Method != "connect" {
			resp := ResponseFrame{
				Type: FrameTypeResponse,
				ID:   req.ID,
				OK:   false,
				Error: &ErrorShape{
					Code:    "INVALID_REQUEST",
					Message: "expected connect method",
				},
			}
			if err := conn.WriteJSON(resp); err != nil {
				return
			}
			return
		}

		// Check password
		var params ConnectParams
		paramsJSON, err := json.Marshal(req.Params)
		if err != nil {
			return
		}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return
		}

		if params.Auth == nil || params.Auth.Password != password {
			resp := ResponseFrame{
				Type: FrameTypeResponse,
				ID:   req.ID,
				OK:   false,
				Error: &ErrorShape{
					Code:    "NOT_LINKED",
					Message: "invalid password",
				},
			}
			if err := conn.WriteJSON(resp); err != nil {
				return
			}
			return
		}

		// Step 3: Send hello-ok response
		helloOk := HelloOk{
			Type:     "hello-ok",
			Protocol: ProtocolVersion,
			Server: ServerInfo{
				Version: "2026.3.1",
				ConnID:  "test-conn-001",
			},
			Features: Features{
				Methods: []string{"health", "agents.list", "sessions.list", "chat.send"},
				Events:  []string{"tick", "chat", "agent", "presence", "shutdown"},
			},
			Snapshot: Snapshot{
				UptimeMs: 86400000,
				SessionDefaults: &SessionDefaults{
					DefaultAgentID: "main",
					MainKey:        "main",
					MainSessionKey: "main:default",
					Scope:          "per-sender",
				},
			},
			Policy: HelloPolicy{
				MaxPayload:       25 * 1024 * 1024,
				MaxBufferedBytes: 50 * 1024 * 1024,
				TickIntervalMs:   30000,
			},
		}

		resp := ResponseFrame{
			Type:    FrameTypeResponse,
			ID:      req.ID,
			OK:      true,
			Payload: mustMarshal(helloOk),
		}
		if err := conn.WriteJSON(resp); err != nil {
			return
		}

		// Step 4: Handle subsequent requests
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var frame RequestFrame
			if err := json.Unmarshal(msg, &frame); err != nil {
				continue
			}

			switch frame.Method {
			case "health":
				if err := conn.WriteJSON(ResponseFrame{
					Type:    FrameTypeResponse,
					ID:      frame.ID,
					OK:      true,
					Payload: mustMarshal(map[string]string{"status": "ok"}),
				}); err != nil {
					return
				}

			case "agents.list":
				result := AgentsListResult{
					DefaultID: "main",
					MainKey:   "main",
					Scope:     "per-sender",
					Agents: []AgentSummary{
						{
							ID:   "main",
							Name: "Main Agent",
							Identity: &AgentIdentity{
								Name:  "Claude",
								Emoji: "🤖",
							},
						},
						{
							ID:   "gemini",
							Name: "Gemini Agent",
							Identity: &AgentIdentity{
								Name: "Gemini",
							},
						},
					},
				}
				if err := conn.WriteJSON(ResponseFrame{
					Type:    FrameTypeResponse,
					ID:      frame.ID,
					OK:      true,
					Payload: mustMarshal(result),
				}); err != nil {
					return
				}

			case "chat.send":
				if err := conn.WriteJSON(ResponseFrame{
					Type:    FrameTypeResponse,
					ID:      frame.ID,
					OK:      true,
					Payload: mustMarshal(map[string]string{"runId": "test-run-001"}),
				}); err != nil {
					return
				}

			default:
				if err := conn.WriteJSON(ResponseFrame{
					Type: FrameTypeResponse,
					ID:   frame.ID,
					OK:   false,
					Error: &ErrorShape{
						Code:    "INVALID_REQUEST",
						Message: "unknown method: " + frame.Method,
					},
				}); err != nil {
					return
				}
			}
		}
	}))
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestClientConnect(t *testing.T) {
	server := mockGateway(t, "test-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "test-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close()

	hello := client.Hello()
	require.NotNil(t, hello)
	assert.Equal(t, ProtocolVersion, hello.Protocol)
	assert.Equal(t, "2026.3.1", hello.Server.Version)
	assert.Equal(t, "test-conn-001", hello.Server.ConnID)
}

func TestClientConnectBadPassword(t *testing.T) {
	server := mockGateway(t, "correct-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "wrong-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")
}

func TestClientHealth(t *testing.T) {
	server := mockGateway(t, "test-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "test-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.Connect(ctx))
	defer client.Close()

	err := client.Health(ctx)
	assert.NoError(t, err)
}

func TestClientListAgents(t *testing.T) {
	server := mockGateway(t, "test-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "test-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.Connect(ctx))
	defer client.Close()

	result, err := client.ListAgents(ctx)
	require.NoError(t, err)
	assert.Equal(t, "main", result.DefaultID)
	assert.Len(t, result.Agents, 2)
	assert.Equal(t, "main", result.Agents[0].ID)
	assert.Equal(t, "Claude", result.Agents[0].Identity.Name)
	assert.Equal(t, "gemini", result.Agents[1].ID)
}

func TestClientChatSend(t *testing.T) {
	server := mockGateway(t, "test-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "test-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.Connect(ctx))
	defer client.Close()

	err := client.ChatSend(ctx, "main:default", "Hello world")
	assert.NoError(t, err)
}

func TestClientRequestTimeout(t *testing.T) {
	server := mockGateway(t, "test-password")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL, "test-password")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.Connect(ctx))
	defer client.Close()

	// Request with already-cancelled context
	cancelledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()

	_, err := client.Request(cancelledCtx, "anything", nil)
	assert.Error(t, err)
}
