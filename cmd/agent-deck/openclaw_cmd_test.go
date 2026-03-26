package main

import "testing"

func TestBuildOpenClawBridgeCommand(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		want    string
	}{
		{
			name:    "simple id remains unquoted",
			agentID: "agent-123",
			want:    "agent-deck openclaw bridge --agent agent-123",
		},
		{
			name:    "spaces are shell-quoted",
			agentID: "agent with spaces",
			want:    "agent-deck openclaw bridge --agent 'agent with spaces'",
		},
		{
			name:    "single quote is escaped safely",
			agentID: "agent'$(whoami)",
			want:    "agent-deck openclaw bridge --agent 'agent'\"'\"'$(whoami)'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildOpenClawBridgeCommand(tc.agentID)
			if got != tc.want {
				t.Fatalf("buildOpenClawBridgeCommand(%q) = %q, want %q", tc.agentID, got, tc.want)
			}
		})
	}
}
