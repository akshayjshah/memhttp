package memhttp

import (
	"context"
	"log"
	"time"
)

type config struct {
	DisableTLS     bool
	DisableHTTP2   bool
	CleanupContext func() (context.Context, context.CancelFunc)
	ErrorLog       *log.Logger
}

// An Option configures a Server.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) { f(cfg) }

// WithoutTLS disables TLS on the server and client.
func WithoutTLS() Option {
	return optionFunc(func(cfg *config) {
		cfg.DisableTLS = true
	})
}

// WithoutHTTP2 disables HTTP/2 on the server and client.
func WithoutHTTP2() Option {
	return optionFunc(func(cfg *config) {
		cfg.DisableHTTP2 = true
	})
}

// WithOptions composes multiple Options into one.
func WithOptions(opts ...Option) Option {
	return optionFunc(func(cfg *config) {
		for _, opt := range opts {
			opt.apply(cfg)
		}
	})
}

// WithCleanupTimeout customizes the default five-second timeout for the
// server's Cleanup method. It's most useful with the memhttptest subpackage.
func WithCleanupTimeout(d time.Duration) Option {
	return optionFunc(func(cfg *config) {
		cfg.CleanupContext = func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), d)
		}
	})
}

// WithErrorLog sets [http.Server.ErrorLog].
func WithErrorLog(l *log.Logger) Option {
	return optionFunc(func(cfg *config) {
		cfg.ErrorLog = l
	})
}
