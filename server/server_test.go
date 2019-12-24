package server

import (
	"golang.org/x/net/websocket"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func ExampleConfig_Create() {
	config := Config{
		Endpoints: []Endpoint{
			{Protocol: "tcp", Address: "127.0.0.1:12345", Path: "/ws"},
		},
		Timeout: 15 * time.Second,
	}

	http.ListenAndServe(":8080", config.Create())
}

func TestConfig_Create(t *testing.T) {
	// setup back
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Error(err)
		return
	}
	defer listener.Close()

	var received = make(chan string, 1)
	go func() {
		defer close(received)
		conn, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		defer conn.Close()

		var sent = make(chan struct{})
		go func() {
			_, _ = conn.Write([]byte("pong"))
			close(sent)
		}()
		var buf [8]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			t.Error(err)
		}
		received <- string(buf[:n])
		<-sent
	}()

	// setup proxy
	config := Config{
		Endpoints: []Endpoint{
			{Protocol: "tcp", Address: listener.Addr().String(), Path: "/ws"},
		},
		Timeout: 15 * time.Second,
	}
	srv := httptest.NewServer(config.Create())
	defer srv.Close()
	ref := strings.ReplaceAll(srv.URL, "http://", "ws://") + "/ws"
	t.Log(ref)
	// setup client
	ws, err := websocket.Dial(ref, "", "http://example.com")
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()

	// make request
	_, err = ws.Write([]byte("ping"))
	if err != nil {
		t.Error(err)
		return
	}

	var buf [8]byte
	n, err := ws.Read(buf[:])
	if err != nil {
		t.Error(err)
		return
	}
	var pong = string(buf[:n])
	var ping string
	select {
	case ping = <-received:
	case <-time.After(5 * time.Second):
		t.Error("timeout")
		return
	}

	if pong != "pong" {
		t.Error("not a pong:", pong)
	}
	if ping != "ping" {
		t.Error("not a ping:", ping)
	}
	t.Logf("Backend got [%v], Client got [%v]", ping, pong)
}
