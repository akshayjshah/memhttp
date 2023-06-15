package memhttptest_test

import (
	"io"
	"net/http"
	"testing"

	"go.akshayshah.org/memhttp/memhttptest"
)

func Example() {
	// Typically, you'd get a *testing.T from your unit test.
	_ = func(t *testing.T) {
		hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "Hello, world!")
		})
		// The server is already running, and it automatically shuts down
		// gracefully when the test ends.
		srv := memhttptest.New(t, hello)
		res, err := srv.Client().Get(srv.URL())
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Error(res.Status)
		}
	}
}
