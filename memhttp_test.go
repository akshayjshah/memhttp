package memhttp_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"go.akshayshah.org/attest"
	"go.akshayshah.org/memhttp"
	"go.akshayshah.org/memhttp/memhttptest"
)

const greeting = "hello world"

type greeter struct{}

func (h *greeter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(greeting))
}

func TestServer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []memhttp.Option
	}{
		{"default", nil},
		{"plaintext", []memhttp.Option{memhttp.WithoutTLS()}},
		{"http1", []memhttp.Option{memhttp.WithoutHTTP2()}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const concurrency = 100
			srv := memhttptest.New(t, &greeter{}, tt.opts...)
			var wg sync.WaitGroup
			start := make(chan struct{})
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					client := srv.Client()
					req, err := http.NewRequestWithContext(
						context.Background(),
						http.MethodGet,
						srv.URL(),
						strings.NewReader(""),
					)
					attest.Ok(t, err)
					<-start
					res, err := client.Do(req)
					attest.Ok(t, err)
					attest.Equal(t, res.StatusCode, http.StatusOK, attest.Continue())
					body, err := io.ReadAll(res.Body)
					attest.Ok(t, err)
					attest.Equal(t, string(body), greeting)
				}()
			}
			close(start)
			wg.Wait()
		})
	}
}

func TestRegisterOnShutdown(t *testing.T) {
	t.Parallel()
	srv, err := memhttp.New(&greeter{})
	attest.Ok(t, err)
	done := make(chan struct{})
	srv.RegisterOnShutdown(func() {
		close(done)
	})
	attest.Ok(t, srv.Shutdown(context.Background()))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("OnShutdown hook didn't fire")
	}
}

func Example() {
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, world!")
	})
	srv, err := memhttp.New(hello)
	if err != nil {
		panic(err)
	}
	defer srv.Close()
	res, err := srv.Client().Get(srv.URL())
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Status)
	// Output:
	// 200 OK
}

func ExampleServer_Transport() {
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, world!")
	})
	srv, err := memhttp.New(hello)
	if err != nil {
		panic(err)
	}
	defer srv.Close()
	transport := srv.Transport()
	transport.IdleConnTimeout = 10 * time.Second
	client := &http.Client{Transport: transport}
	res, err := client.Get(srv.URL())
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Status)
	// Output:
	// 200 OK
}

func ExampleServer_Client() {
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, world!")
	})
	srv, err := memhttp.New(hello)
	if err != nil {
		panic(err)
	}
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 10 * time.Second
	res, err := client.Get(srv.URL())
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Status)
	// Output:
	// 200 OK
}

func ExampleServer_Shutdown() {
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, world!")
	})
	srv, err := memhttp.New(hello)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
}
