package app

import (
	"context"
	"net/http"
)

type Server interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type HTTPServer struct {
	Srv *http.Server
}

func (h *HTTPServer) ListenAndServe() error {
	return h.Srv.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.Srv.Shutdown(ctx)
}
