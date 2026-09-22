package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// A stop signal lets a request already in flight finish, then refuses new ones.
func TestServeHTTPDrainsInFlight(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := "http://" + ln.Addr().String()
	entered, release := make(chan struct{}), make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		io.WriteString(w, "done")
	})}

	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serveHTTP(ctx, srv, ln, 5*time.Second) }()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		got <- result{string(b), err}
	}()

	<-entered
	stop()
	select {
	case err := <-served:
		t.Fatalf("server returned with a request in flight: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	if r := <-got; r.err != nil || r.body != "done" {
		t.Fatalf("in-flight request: body=%q err=%v", r.body, r.err)
	}
	if err := <-served; err != nil {
		t.Fatalf("clean shutdown returned %v", err)
	}
	if _, err := http.Get(url); err == nil {
		t.Fatal("server still accepts connections after shutdown")
	}
}

// A request that outlives the timeout makes the shutdown report it instead of
// hanging.
func TestServeHTTPShutdownTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
	})}
	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serveHTTP(ctx, srv, ln, 50*time.Millisecond) }()
	go http.Get("http://" + ln.Addr().String())

	<-entered
	stop()
	select {
	case err := <-served:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want deadline exceeded, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown hung past its timeout")
	}
}

// A listener that fails is reported without waiting for a stop signal.
func TestServeHTTPServeError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln.Close()
	done := make(chan error, 1)
	go func() { done <- serveHTTP(context.Background(), &http.Server{}, ln, time.Second) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("want serve error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve error not reported")
	}
}
