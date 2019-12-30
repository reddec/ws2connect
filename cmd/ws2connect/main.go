package main

import (
	"context"
	"errors"
	auth "github.com/abbot/go-http-auth"
	"github.com/foomo/htpasswd"
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
	Authorization    struct {
		Kind     string `short:"k" long:"kind" env:"KIND" description:"Authorization kind" default:"none" choice:"none" choice:"basic" choice:"digest"`
		Realm    string `long:"realm" env:"REALM" description:"Name of authorization zone" default:"Restricted zone"`
		Htpasswd string `short:"p" long:"htpasswd" env:"HTPASSWD" description:"Path to htpasswd (bcrypt or sha) file for user authorization"`
	} `group:"Authorization" namespace:"auth" env-namespace:"AUTH"`
	Args struct {
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

	mux := http.NewServeMux()

	if len(eps) > 0 {
		cfg := server.Config{
			Endpoints: eps,
			Timeout:   config.Timeout,
		}
		mux.Handle("/", cfg.Create())
	}
	if config.Dynamic != "" {
		cfg := server.DynamicConfig{Timeout: config.Timeout}
		mux.Handle(config.Dynamic, http.StripPrefix(config.Dynamic, cfg.Create()))
	}

	var handler http.Handler = mux
	if config.Authorization.Kind != "none" {
		wrapped, err := wrapAuthProxy(handler)
		if err != nil {
			return err
		}
		handler = wrapped
	}
	if config.CORS {
		handler = cors.AllowAll().Handler(handler)
	}

	srv := http.Server{
		Addr:    config.Binding,
		Handler: handler,
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

func wrapAuthProxy(handler http.Handler) (http.Handler, error) {
	authData, err := htpasswd.ParseHtpasswdFile(config.Authorization.Htpasswd)
	if err != nil {
		return nil, err
	}
	var wrapped http.Handler
	switch config.Authorization.Kind {
	case "basic":
		wrapped = auth.NewBasicAuthenticator(config.Authorization.Realm, lookupUserMap(authData)).Wrap(wrapAuthRequest(handler))
	case "digest":
		wrapped = auth.NewDigestAuthenticator(config.Authorization.Realm, lookupUserMap(authData)).Wrap(wrapAuthRequest(handler))
	default:
		return nil, errors.New("unknown authorization kind " + config.Authorization.Kind)
	}
	return wrapped, nil
}

func wrapAuthRequest(handler http.Handler) auth.AuthenticatedHandlerFunc {
	return func(writer http.ResponseWriter, request *auth.AuthenticatedRequest) {
		handler.ServeHTTP(writer, &request.Request)
	}
}

func lookupUserMap(authData map[string]string) auth.SecretProvider {
	return func(user, realm string) string {
		passwd, ok := authData[user]
		if !ok {
			return ""
		}
		return passwd
	}
}
