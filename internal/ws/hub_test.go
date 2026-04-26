package ws_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"github.com/Leumas-LSN/benchere/internal/ws"
)

var upgrader = gorillaws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func TestHub_Broadcast(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	// Channel to signal when client is registered
	clientRegistered := make(chan bool, 1)
	
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Register the connection with the hub
		hub.Register(conn)
		clientRegistered <- true
		
		// Keep the connection open but don't read from it
		// The hub will write broadcast messages to it
		go func() {
			defer hub.Unregister(conn)
			defer conn.Close()
			// Just keep the connection alive, don't read
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					return
				}
			}
		}()
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:] + "/"
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Wait for registration
	<-clientRegistered
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event
	hub.Broadcast(ws.Event{
		Type:    ws.EventJobStatus,
		JobID:   "j1",
		Payload: ws.MustMarshal(ws.JobStatusPayload{Status: "running", Phase: "provisioning"}),
	})

	// The client should receive the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var e ws.Event
	if err := json.Unmarshal(msg, &e); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if e.Type != ws.EventJobStatus {
		t.Errorf("got type %q, want %q", e.Type, ws.EventJobStatus)
	}
	if e.JobID != "j1" {
		t.Errorf("got job_id %q, want j1", e.JobID)
	}
}
