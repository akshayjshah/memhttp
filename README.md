memhttp
=======

[![Build](https://github.com/akshayjshah/memhttp/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/akshayjshah/memhttp/actions/workflows/ci.yaml)
[![Report Card](https://goreportcard.com/badge/go.akshayshah.org/memhttp)](https://goreportcard.com/report/go.akshayshah.org/memhttp)
[![GoDoc](https://pkg.go.dev/badge/go.akshayshah.org/memhttp.svg)](https://pkg.go.dev/go.akshayshah.org/memhttp)


`memhttp` provides a full `net/http` server and client that communicate over
in-memory pipes rather than the network. This is often useful in tests, where
you want to avoid localhost networking but don't want to stub out all the
complexity of HTTP.

Occasionally, it's also useful in production code: if you're planning to split a
monolithic application into microservices, you can first use `memhttp` to
simulate the split. This allows you to rewrite local function calls as HTTP
calls (complete with serialization, compression, and middleware) while
retaining the ability to quickly change service boundaries.

In particular, `memhttp` pairs well with [`connect-go`][connect-go] RPC
servers.

## Installation

```
go get go.akshayshah.org/memhttp
```

## Usage

In-memory HTTP is most common in tests, so most users will be best served by
the `memhttptest` subpackage:

```go
func TestServer(t *testing.T) {
  hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, world!")
  })
  // The server starts automatically, and it shuts down gracefully when the
  // test ends. Startup and shutdown errors fail the test.
  //
  // By default, servers and clients use TLS and support HTTP/2.
  srv := memhttptest.New(t, hello)
  res, err := srv.Client().Get(srv.URL())
  if err != nil {
    t.Fatal(err)
  }
  if res.StatusCode != http.StatusOK {
    t.Error(res.Status)
  }
}
```

## Status: Unstable

This module is unstable, with a stable release expected before the end of 2023.
It supports the [two most recent major releases][go-support-policy] of Go.

Within those parameters, `memhttp` follows semantic versioning. 

## Legal

Offered under the [MIT license][license].

[go-support-policy]: https://golang.org/doc/devel/release#policy
[license]: https://github.com/akshayjshah/memhttp/blob/main/LICENSE
[connect-go]: https://github.com/bufbuild/connect-go
