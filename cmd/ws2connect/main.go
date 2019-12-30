package main

import (
	"context"
	"github.com/jessevdk/go-flags"
	"github.com/reddec/ws2connect/server"
	"github.com/rs/cors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var version string = "dev"

var config struct {
	Binding          string        `short:"b" long:"binding" env:"BINDING" description:"HTTP binding address" default:":8080"`
	Timeout          time.Duration `short:"t" long:"timeout" env:"TIMEOUT" description:"Backend connection timeout" default:"15s"`
	GracefulShutdown time.Duration `long:"graceful-shutdown" env:"GRACEFUL_SHUTDOWN" description:"Delay before server shutdown" default:"15s"`
	TLS              bool          `long:"tls" env:"TLS" description:"Enable HTTPS serving with TLS"`
	CertFile         string        `long:"cert-file" env:"CERT_FILE" description:"Path to certificate for TLS" default:"server.crt"`
	KeyFile          string        `long:"key-file" env:"KEY_FILE" description:"Path to private key for TLS" default:"server.key"`
	Quiet            bool          `short:"q" long:"quiet" env:"QUIET" description:"Disable logging"`
	CORS             bool          `long:"cors" env:"CORS" description:"Enable CORS for HTTP server"`
	Dynamic          string        `short:"d" long:"dynamic" env:"DYNAMIC" description:"Dynamic endpoint mapping path"`
	Args             struct {
		Endpoint map[string]string `positional-arg-name:"endpoints" env:"ENDPOINT" description:"Endpoint mapping (/path:address)" default:"/:127.0.0.1:12345" env-delim:";"`
	} `positional-args:"yes"`
}

func main() {
	parser := flags.NewParser(&config, flags.Default)
	parser.LongDescription = "Expose any TCP service over websocket\nAuthor: Baryshnikov Aleksandr <dev@baryshnikov.net>\nVersion: " + version
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
	if config.Quiet {
		log.SetOutput(ioutil.Discard)
	}
	err = run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var eps = make([]server.Endpoint, 0, len(config.Args.Endpoint))
	for path, addr := range config.Args.Endpoint {
		eps = append(eps, server.Endpoint{
			Path:     path,
			Address:  addr,
			Protocol: "tcp",
		})
		log.Println(path, "->", addr, "(tcp)")
	}

	cfg := server.Config{
		Endpoints: eps,
		Timeout:   config.Timeout,
	}

	var handler = cfg.Create()
	if config.CORS {
		handler = cors.AllowAll().Handler(handler)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	if config.Dynamic != "" {
		mux.Handle(config.Dynamic, http.StripPrefix(config.Dynamic, handler))
	}

	srv := http.Server{
		Addr:    config.Binding,
		Handler: mux,
	}
	log.Println("server started on", config.Binding)
	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Kill, os.Interrupt)
		<-c
		ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdown)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	if config.TLS {
		return srv.ListenAndServeTLS(config.CertFile, config.KeyFile)
	}
	return srv.ListenAndServe()
}
