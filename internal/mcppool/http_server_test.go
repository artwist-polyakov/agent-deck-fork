package mcppool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPServer_ExternalServer(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create HTTPServer pointing to external server (no command)
	server := NewHTTPServer(context.Background(), "test", ts.URL, ts.URL, "", nil, nil, time.Second)

	// Start should detect the external server
	if err := server.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Should be running
	if !server.IsRunning() {
		t.Error("Expected server to be running")
	}

	// Should NOT be started by us
	if server.StartedByUs() {
		t.Error("Expected StartedByUs() to be false for external server")
	}

	// Stop should work
	if err := server.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Should be stopped
	if server.GetStatus() != StatusStopped {
		t.Errorf("Expected status Stopped, got %v", server.GetStatus())
	}
}

func TestHTTPPool_RegisterExternal(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	pool := NewHTTPPool(context.Background())

	// Register external server
	if err := pool.RegisterExternal("test", ts.URL); err != nil {
		t.Fatalf("RegisterExternal() failed: %v", err)
	}

	// Should be running
	if !pool.IsRunning("test") {
		t.Error("Expected server to be running after registration")
	}

	// Get server
	server := pool.GetServer("test")
	if server == nil {
		t.Fatal("GetServer() returned nil")
	}

	// Should NOT be started by us
	if server.StartedByUs() {
		t.Error("Expected StartedByUs() to be false")
	}

	// List servers
	servers := pool.ListServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	// Shutdown
	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
}

func TestHTTPPool_StartStop(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	pool := NewHTTPPool(context.Background())

	// Start with external URL (no command means it will discover external)
	err := pool.Start("test", ts.URL, ts.URL, "", nil, nil, time.Second)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Should be running
	if !pool.IsRunning("test") {
		t.Error("Expected server to be running")
	}

	// Get running count
	if count := pool.GetRunningCount(); count != 1 {
		t.Errorf("Expected 1 running, got %d", count)
	}

	// Shutdown
	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
}
