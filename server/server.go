package server

import (
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// Endpoint definition that contains remote address (with port) and protocol (supported by Go Dial function)
type Endpoint struct {
	Path     string // front HTTP binding path (ex: /ws)
	Address  string // remote address with host and port (if applicable)
	Protocol string // protocol type: tcp, udp, unix
}

// Configuration for HTTP WS reverse proxy to multiple backend
type Config struct {
	Endpoints []Endpoint    // endpoint configuration (to where connection will be established)
	Timeout   time.Duration // connection timeout
}

// Create HTTP handler with internal mapping of exported path and remote addresses
func (c Config) Create() http.Handler {
	mux := http.NewServeMux()
	for _, ep := range c.Endpoints {
		mux.HandleFunc(ep.Path, func(writer http.ResponseWriter, request *http.Request) {
			log.Println("incoming request from", request.RemoteAddr, "will be mapped to", ep.Address, "(", ep.Protocol, ")")
			c.makeProxy(ep, writer, request)
		})
	}
	return mux
}

func (c Config) makeProxy(ep Endpoint, writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	conn, err := net.DialTimeout(ep.Protocol, ep.Address, c.Timeout)
	if err != nil {
		log.Println("connection failed to remote address", ep.Address, "(", ep.Protocol, "):", err)
		http.Error(writer, "failed to connect", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	websocket.Handler(func(ws *websocket.Conn) {
		done := make(chan struct{})
		go func() {
			io.Copy(ws, conn)
			ws.Close()
			conn.Close()
			close(done)
		}()
		io.Copy(conn, ws)
		conn.Close()
		ws.Close()
		<-done
	}).ServeHTTP(writer, request)
	log.Println("connection", request.RemoteAddr, "->", ep.Address, "(", ep.Protocol, ")", "closed")
}
