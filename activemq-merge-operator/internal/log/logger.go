// Package log handle log configuration
package log

import (
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.pnet.ch/linux/go/slogcfg"
)

// Config provides possibility to configure logger
type Config struct {
	WithoutCaller     bool               `env:"WITHOUT_CALLER" env-description:"do not annotate logs with the calling function's file name and number"`
	WithoutTimestamps bool               `env:"WITHOUT_TIMESTAMPS" env-description:"do not log any timestamps"`
	Debug             bool               `env:"DEBUG" env-description:"log debug information"`
	Format            slogcfg.FormatType `env:"FORMAT" env-description:"configures the log format (text or json)" env-default:"text"`
}

// New returns a new slog.Handler instance based on the provided config
func (c Config) New(reg prometheus.Registerer) slog.Handler {
	logOpts := options(c)
	logOpts = append(logOpts, slogcfg.WithPrometheus(reg))

	return slogcfg.MustNewHandler(os.Stderr, logOpts...)
}

func options(c Config) []slogcfg.Option {
	opts := []slogcfg.Option{
		slogcfg.WithDebug(c.Debug),
	}

	if c.WithoutCaller {
		opts = append(opts, slogcfg.WithAddSource(!c.WithoutCaller))
	}

	if c.WithoutTimestamps {
		opts = append(opts, slogcfg.WithoutKeys(slog.TimeKey))
	}

	return append(opts, slogcfg.WithFormat(c.Format))
}
