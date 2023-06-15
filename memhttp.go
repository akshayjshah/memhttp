// Package memhttp provides an in-memory HTTP server and client. For
// testing-specific adapters, see the memhttptest subpackage.
package memhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Server is a net/http server that uses in-memory pipes instead of TCP. By
// default, it has TLS enabled and supports HTTP/2. It otherwise uses the same
// configuration as the zero value of [http.Server].
type Server struct {
	server         *http.Server
	listener       *memoryListener
	certificate    *x509.Certificate // for client
	url            string
	disableHTTP2   bool
	serveErr       chan error
	cleanupContext func() (context.Context, context.CancelFunc)
}

// New constructs and starts a Server.
func New(handler http.Handler, opts ...Option) (*Server, error) {
	var cfg config
	WithCleanupTimeout(5 * time.Second).apply(&cfg)
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	mlis := &memoryListener{
		conns:  make(chan net.Conn),
		closed: make(chan struct{}),
	}
	var lis net.Listener = mlis
	server := &http.Server{Handler: handler}

	var clientCert *x509.Certificate
	if !cfg.DisableTLS {
		srvCert, err := tls.X509KeyPair(_cert, _key)
		if err != nil {
			return nil, fmt.Errorf("create x509 key pair: %v", err)
		}
		protos := []string{"h2"}
		if cfg.DisableHTTP2 {
			protos = []string{"http/1.1"}
		}
		server.TLSConfig = &tls.Config{
			NextProtos:   protos,
			Certificates: []tls.Certificate{srvCert},
		}
		clientCert, err = x509.ParseCertificate(server.TLSConfig.Certificates[0].Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("parse x509 certificate: %v", err)
		}
		lis = tls.NewListener(mlis, server.TLSConfig)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(lis)
	}()

	scheme := "https://"
	if cfg.DisableTLS {
		scheme = "http://"
	}
	return &Server{
		server:         server,
		listener:       mlis,
		certificate:    clientCert,
		url:            scheme + mlis.Addr().String(),
		disableHTTP2:   cfg.DisableHTTP2,
		serveErr:       serveErr,
		cleanupContext: cfg.CleanupContext,
	}, nil
}

// Transport returns an [http.Transport] configured to use in-memory pipes
// rather than TCP, disable automatic compression, trust the server's TLS
// certificate (if any), and use HTTP/2 (if the server supports it).
//
// Callers may reconfigure the returned Transport without affecting other
// transports or clients.
func (s *Server) Transport() *http.Transport {
	transport := &http.Transport{
		DialContext:        s.listener.DialContext,
		DisableCompression: true,
	}
	if s.certificate != nil {
		pool := x509.NewCertPool()
		pool.AddCert(s.certificate)
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		transport.ForceAttemptHTTP2 = !s.disableHTTP2
	}
	return transport
}

// Client returns an [http.Client] configured to use in-memory pipes rather
// than TCP, disable automatic compression, trust the server's TLS certificate
// (if any), and use HTTP/2 (if the server supports it).
//
// Callers may reconfigure the returned client without affecting other clients.
func (s *Server) Client() *http.Client {
	return &http.Client{Transport: s.Transport()}
}

// URL returns the server's URL.
func (s *Server) URL() string {
	return s.url
}

// Close immediately shuts down the server. To shut down the server without
// interrupting in-flight requests, use Shutdown.
func (s *Server) Close() error {
	if err := s.server.Close(); err != nil {
		return err
	}
	return s.listenErr()
}

// Shutdown gracefully shuts down the server, without interrupting any active
// connections. See [http.Server.Shutdown] for details.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	return s.listenErr()
}

// Cleanup calls Shutdown with a five second timeout. To customize the timeout,
// use WithCleanupTimeout.
//
// Cleanup is primarily intended for use in tests. If you find yourself using
// it, you may want to use the memhttptest package instead.
func (s *Server) Cleanup() error {
	ctx, cancel := s.cleanupContext()
	defer cancel()
	return s.Shutdown(ctx)
}

// RegisterOnShutdown registers a function to call on Shutdown. It's often used
// to cleanly shut down connections that have been hijacked. See
// [http.Server.RegisterOnShutdown] for details.
func (s *Server) RegisterOnShutdown(f func()) {
	s.server.RegisterOnShutdown(f)
}

func (s *Server) listenErr() error {
	if err := <-s.serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type memoryListener struct {
	conns  chan net.Conn
	once   sync.Once
	closed chan struct{}
}

// Accept implements net.Listener.
func (l *memoryListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, errors.New("listener closed")
	}
}

// Close implements net.Listener.
func (l *memoryListener) Close() error {
	l.once.Do(func() {
		close(l.closed)
	})
	return nil
}

// Addr implements net.Listener.
func (l *memoryListener) Addr() net.Addr {
	return &memoryAddr{}
}

// DialContext is the type expected by http.Transport.DialContext.
func (l *memoryListener) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, errors.New("listener closed")
	default:
	}
	server, client := net.Pipe()
	l.conns <- server
	return client, nil
}

type memoryAddr struct{}

// Network implements net.Addr.
func (*memoryAddr) Network() string { return "memory" }

// String implements io.Stringer, returning a value that matches the
// certificates used by net/http/httptest.
func (*memoryAddr) String() string { return "example.com" }
