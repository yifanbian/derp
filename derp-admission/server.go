package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"tailscale.com/tailcfg"
)

const (
	maxRequestBodyBytes = 2048
	shutdownTimeout     = 5 * time.Second
)

func newHandler(nodes *NodeCache, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tailcfg.DERPAdmitClientRequest
		if err := decodeAdmitRequest(w, r, &req); err != nil {
			logger.Debug("invalid admission request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		body, err := json.Marshal(tailcfg.DERPAdmitClientResponse{
			Allow: nodes.Contains(req.NodePublic),
		})
		if err != nil {
			logger.Error("encode admission response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(body); err != nil {
			logger.Debug("write admission response", "error", err)
		}
	})
}

func decodeAdmitRequest(w http.ResponseWriter, r *http.Request, req *tailcfg.DERPAdmitClientRequest) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	if err := dec.Decode(req); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func serve(ctx context.Context, addr string, handler http.Handler) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       time.Minute,
	}

	result := make(chan error, 1)
	go func() { result <- srv.Serve(ln) }()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			closeErr := srv.Close()
			<-result
			return errors.Join(fmt.Errorf("shutdown: %w", err), closeErr)
		}
		<-result
		return nil
	}
}
