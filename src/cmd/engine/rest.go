package engine

import (
	"fmt"
	"github.com/rs/cors"
	"net"
	"net/http"
	"time"

	"github.com/onflow/flow-go/engine"
	accessproto "github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/rs/zerolog"
)

// RESTConfig defines the configurable options for the REST server.
type RESTConfig struct {
	ListenAddr string
}

// REST implements a REST server for the API service node
// The RPC router reads from the channel and forwards requests to the gRPC AccessAPIServer servers
type REST struct {
	unit   *engine.Unit
	log    zerolog.Logger
	server *http.Server // the REST server
	config RESTConfig

	proxy accessproto.AccessAPIServer
}

// NewRESTEngine returns a new REST engine.
func NewRESTEngine(log zerolog.Logger, config RESTConfig, proxy accessproto.AccessAPIServer) (*REST, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy not set")
	}

	log = log.With().Str("engine", "rest").Logger()

	httpServer, err := newHTTPServer(proxy, config.ListenAddr, log)
	if err != nil {
		return nil, err
	}

	eng := &REST{
		log:    log,
		unit:   engine.NewUnit(),
		server: httpServer,
		config: config,
	}

	return eng, nil
}

// newHTTPServer returns an HTTP server initialized with a gRPC API router
// TODO handle select, etc. backend functions of the access node
func newHTTPServer(upstream accessproto.AccessAPIServer, listenAddress string, logger zerolog.Logger) (*http.Server, error) {
	router, err := NewRouter(upstream, logger)
	if err != nil {
		return nil, err
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodOptions,
			http.MethodHead},
	})

	return &http.Server{
		Addr:         listenAddress,
		Handler:      c.Handler(router),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}, nil
}

// Ready starts listening to an HTTP port and returns a ready channel.
// The channel is closed once the engine has fully started.
// The REST engine is ready, when the HTTP server has successfully started.
func (e *REST) Ready() <-chan struct{} {
	e.unit.Launch(e.serve)
	return e.unit.Ready()
}

// Done returns a done channel that is closed once the engine has fully stopped.
// It sends a signal to stop the HTTP server, then closes the channel.
func (e *REST) Done() <-chan struct{} {
	return e.unit.Done(func() {
		err := e.server.Close()
		e.log.Err(err).Msg("failed to stop server")
	})
}

// serve implements the REST gRPC server.
// When this function returns, the server is considered ready.
func (e *REST) serve() {
	e.log.Info().Msgf("starting server on address %s", e.config.ListenAddr)

	l, err := net.Listen("tcp", e.config.ListenAddr)
	if err != nil {
		e.log.Err(err).Msg("failed to start server")
		return
	}

	err = e.server.Serve(l)
	if err != nil {
		e.log.Err(err).Msg("fatal error in server")
	}
}
