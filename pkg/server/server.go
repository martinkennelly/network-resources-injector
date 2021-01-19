package server

import (
	"context"
	"net/http"
	"time"
)

// Server start and stop HTTP server - helps unit tests mocking of HTTP server
type Server interface {
	Start() error
	Stop(timeout time.Duration) error
}

type server struct {
	httpServer *http.Server
}

// Start wraps around package http ListenAndServeTLS and returns any error. Helps unit testing
func (srv *server) Start() error {
	return srv.httpServer.ListenAndServeTLS("", "")
}

// Stop wraps around package http Shutdown limited in time by timeout arg to and returns any error. Helps unit testing
func (srv *server) Stop(to time.Duration) error {
	srv.httpServer.SetKeepAlivesEnabled(false)
	serverCtx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()
	return srv.httpServer.Shutdown(serverCtx)
}
