package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func newTestLogger() *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: false,
	})
}

func TestNewIPCServer(t *testing.T) {
	t.Parallel()

	t.Run("creates server with valid socket path", func(t *testing.T) {
		t.Parallel()
		socketPath := filepath.Join(t.TempDir(), "test.sock")
		logger := newTestLogger()

		srv, err := NewIPCServer(socketPath, logger)
		if err != nil {
			t.Fatalf("NewIPCServer failed: %v", err)
		}
		defer srv.Stop()

		// Verify socket exists with correct permissions.
		info, err := os.Stat(socketPath)
		if err != nil {
			t.Fatalf("socket not created: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("socket permissions = %o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("fails with empty socket path", func(t *testing.T) {
		t.Parallel()
		_, err := NewIPCServer("", newTestLogger())
		if err == nil {
			t.Error("expected error for empty socket path")
		}
	})

	t.Run("removes stale socket", func(t *testing.T) {
		t.Parallel()
		socketPath := filepath.Join(t.TempDir(), "stale.sock")

		// Create a stale file.
		if err := os.WriteFile(socketPath, []byte("stale"), 0644); err != nil {
			t.Fatalf("creating stale file: %v", err)
		}

		srv, err := NewIPCServer(socketPath, newTestLogger())
		if err != nil {
			t.Fatalf("NewIPCServer failed: %v", err)
		}
		defer srv.Stop()
	})
}

func TestIPCServer_PingMethod(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "ping.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	// Give server time to start.
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send ping request.
	req := RPCRequest{Method: "ping", ID: 1}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Read response.
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("response ID = %d, want 1", resp.ID)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %T", resp.Result)
	}
	if pong, _ := result["pong"].(bool); !pong {
		t.Error("expected pong: true")
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_StatusMethod(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "status.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	srv.SetPendingCount(5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send status request.
	req := RPCRequest{Method: "status", ID: 2}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %T", resp.Result)
	}

	if pending, _ := result["pending_count"].(float64); pending != 5 {
		t.Errorf("pending_count = %v, want 5", result["pending_count"])
	}
	if _, ok := result["uptime_seconds"]; !ok {
		t.Error("expected uptime_seconds in status")
	}
	if _, ok := result["active_sessions"]; !ok {
		t.Error("expected active_sessions in status")
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_NotifyMethod(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "notify.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send notify request.
	params, _ := json.Marshal(NotifyParams{Type: "test_event", Payload: map[string]string{"key": "value"}})
	req := RPCRequest{Method: "notify", Params: params, ID: 3}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %T", resp.Result)
	}
	if sent, _ := result["sent"].(bool); !sent {
		t.Error("expected sent: true")
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_NotifyMethod_MissingType(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "notify-missing.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send notify without type.
	params, _ := json.Marshal(NotifyParams{Payload: "data"})
	req := RPCRequest{Method: "notify", Params: params, ID: 4}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error for missing type")
	}
	if resp.Error != nil && resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeInvalidParams)
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_MethodNotFound(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "unknown.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	req := RPCRequest{Method: "unknown_method", ID: 5}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}
	if resp.Error != nil && resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeMethodNotFound)
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_ParseError(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "parse.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON.
	if _, err := conn.Write([]byte("not valid json\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected parse error")
	}
	if resp.Error != nil && resp.Error.Code != ErrCodeParse {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeParse)
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_Subscribe(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "subscribe.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Subscribe.
	req := RPCRequest{Method: "subscribe", ID: 6}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %T", resp.Result)
	}
	if subscribed, _ := result["subscribed"].(bool); !subscribed {
		t.Error("expected subscribed: true")
	}
	if _, ok := result["subscription_id"]; !ok {
		t.Error("expected subscription_id in response")
	}

	// Broadcast an event.
	srv.BroadcastEvent("test_event", map[string]string{"msg": "hello"})

	// Give time for event delivery.
	time.Sleep(50 * time.Millisecond)

	// Set read deadline to avoid hanging.
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

	if scanner.Scan() {
		var eventMsg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &eventMsg); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		event, ok := eventMsg["event"].(map[string]any)
		if !ok {
			t.Fatalf("event not found in message: %v", eventMsg)
		}
		if event["type"] != "test_event" {
			t.Errorf("event type = %v, want test_event", event["type"])
		}
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_MultipleClients(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "multi.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Connect multiple clients.
	const numClients = 5
	conns := make([]net.Conn, numClients)

	for i := range numClients {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("dial client %d failed: %v", i, err)
		}
		conns[i] = conn
	}

	// Each client sends ping.
	for i, conn := range conns {
		req := RPCRequest{Method: "ping", ID: int64(i + 1)}
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			t.Fatalf("write client %d failed: %v", i, err)
		}
	}

	// Each client reads response.
	for i, conn := range conns {
		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			t.Fatalf("no response for client %d", i)
		}

		var resp RPCResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response %d: %v", i, err)
		}
		if resp.Error != nil {
			t.Errorf("client %d error: %v", i, resp.Error)
		}
	}

	// Cleanup.
	for _, conn := range conns {
		conn.Close()
	}

	cancel()
	_ = srv.Stop()
}

func TestIPCServer_GracefulShutdown(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "shutdown.sock")
	srv, err := NewIPCServer(socketPath, newTestLogger())
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Connect a client.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Cancel context and stop server.
	cancel()
	if err := srv.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify socket is cleaned up.
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("socket file should be removed after stop")
	}

	// Server should exit cleanly.
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("server did not exit in time")
	}
}
