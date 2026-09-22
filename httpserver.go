package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"time"
)

// shutdownTimeout bounds how long in-flight requests may run after a stop signal.
const shutdownTimeout = 10 * time.Second

// serveHTTP serves on ln until ctx is done, then stops accepting connections and
// waits up to timeout for in-flight requests to finish. It returns an error only
// if serving failed or the requests did not finish in time.
func serveHTTP(ctx context.Context, srv *http.Server, ln net.Listener, timeout time.Duration) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	sctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// httpAddr resolves the listen address the way gin's Engine.Run does: an empty
// address means $PORT, else :8080.
func httpAddr(addr string) string {
	if addr != "" {
		return addr
	}
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}
