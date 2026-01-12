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
	srv *http.Server
}

func (h *HTTPServer) ListenAndServe() error {
	return h.srv.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
