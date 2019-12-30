package server

import (
	"crypto/tls"
	"crypto/x509"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
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
			makeProxy(ep, c.Timeout, writer, request)
		})
	}
	return mux
}

func makeProxy(ep Endpoint, timeout time.Duration, writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	conn, err := ep.dial(timeout)
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

func (ep Endpoint) dial(timeout time.Duration) (net.Conn, error) {
	if ep.Protocol == "tls" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		dialer := &net.Dialer{
			Timeout:   timeout,
			KeepAlive: 1,
		}
		return tls.DialWithDialer(dialer, "tcp", ep.Address, &tls.Config{RootCAs: pool})
	}
	return net.DialTimeout(ep.Protocol, ep.Address, timeout)
}

// Dynamic configuration for HTTP WS reverse proxy to multiple backend.
//
// Mapping to /<address:port>/<protocol:tls|tcp|udp>
//
type DynamicConfig struct {
	Timeout time.Duration // connection timeout
}

// Create HTTP handler with dynamic mapping to remote addresses
func (c DynamicConfig) Create() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		parts := strings.Split(request.URL.Path, "/")
		if len(parts) != 2 {
			http.Error(writer, "expected path /<address>/<protocol> but got "+request.URL.Path, http.StatusBadRequest)
			return
		}

		address, protocol := parts[0], parts[1]

		log.Println("incoming request from", request.RemoteAddr, "will be mapped to", address, "(", protocol, ")")

		makeProxy(Endpoint{
			Address:  address,
			Protocol: protocol,
		}, c.Timeout, writer, request)
	})
	return mux
}
