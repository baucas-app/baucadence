package logger

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/rs/zerolog"
)

var once sync.Once
var logger zerolog.Logger

// Define a key type to avoid context key collisions
type contextKey string

const loggerKey contextKey = "logger"

func Get() *zerolog.Logger {
	once.Do(func() {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

		logLevel := cfg.LogLevel()

		logger = zerolog.New(os.Stdout).
			Level(zerolog.Level(logLevel)).
			With().
			Timestamp().
			// Caller().
			Logger()
	})
	return &logger
}

// injects the logger into context
func Inject(r *http.Request, l *zerolog.Logger) *http.Request {
	ctx := context.WithValue(r.Context(), loggerKey, l)
	r = r.WithContext(ctx)
	return r
}

func NewContext(l *zerolog.Logger) context.Context {
	ctx := context.WithValue(context.Background(), loggerKey, l)
	return ctx
}

// retrieves the logger from context
func FromContext(ctx context.Context) *zerolog.Logger {
	logger, ok := ctx.Value(loggerKey).(*zerolog.Logger)
	if !ok || logger == nil {
		defaultLogger := zerolog.New(os.Stdout)
		return &defaultLogger
	}
	return logger
}
