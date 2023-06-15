// Package memhttptest adapts the basic memhttp server to be more convenient for
// tests.
package memhttptest

import (
	"log"
	"net/http"
	"testing"

	"go.akshayshah.org/memhttp"
)

// New constructs a [memhttp.Server] with defaults suitable for tests: it logs
// runtime errors to the provided testing.TB, and it automatically shuts down
// the server when the test completes. Startup and shutdown errors fail the
// test.
//
// To customize the server, use any [memhttp.Option]. In particular, it may be
// necessary to customize the shutdown timeout with
// [memhttp.WithCleanupTimeout].
func New(tb testing.TB, h http.Handler, opts ...memhttp.Option) *memhttp.Server {
	tb.Helper()
	logger := log.New(&tbWriter{tb}, "" /* prefix */, log.Lshortfile)
	s, err := memhttp.New(
		h,
		memhttp.WithErrorLog(logger),
		memhttp.WithOptions(opts...),
	)
	if err != nil {
		tb.Fatalf("start in-memory HTTP server: %v", err)
	}
	tb.Cleanup(func() {
		if err := s.Cleanup(); err != nil {
			tb.Error(err)
		}
	})
	return s
}

type tbWriter struct {
	tb testing.TB
}

func (w *tbWriter) Write(bs []byte) (int, error) {
	w.tb.Log(string(bs))
	return len(bs), nil
}
