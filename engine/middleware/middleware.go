package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

type RequestIDHook struct{}

func (h RequestIDHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if ctx := e.GetCtx(); ctx != nil {
		if reqID, ok := ctx.Value("requestID").(string); ok {
			e.Str("request_id", reqID)
		}
	}
}

const requestIDKey MiddlwareContextKey = "requestID"

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateRequestID() string {
	const length = 8 // ~0.23% chance of collision in 1M requests
	id := make([]byte, length)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		id[i] = base62Chars[n.Int64()]
	}
	return string(id)
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GenerateRequestID()
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)

		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}

// Logger logs requests and injects a request-scoped logger with a request ID into the context.
func Logger(baseLogger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			loggerCtx := baseLogger.With().Str("request_id", reqID)

			for key, values := range r.URL.Query() {
				if strings.Contains(strings.ToLower(key), "password") {
					continue
				}
				if len(values) > 0 {
					loggerCtx = loggerCtx.Str(fmt.Sprintf("query.%s", key), values[0])
				}
			}

			l := loggerCtx.Logger()

			// Inject logger into context
			r = logger.Inject(r, &l)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()
			defer func() {
				t2 := time.Now()
				if rec := recover(); rec != nil {
					l.Error().
						Str("type", "error").
						Timestamp().
						Interface("recover_info", rec).
						Bytes("debug_stack", debug.Stack()).
						Msg("log system error")
					utils.WriteError(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				pathS := strings.Split(r.URL.Path, "/")
				msg := fmt.Sprintf("Received %s %s - Responded with %d in %.2fms",
					r.Method, r.URL.Path, ww.Status(), float64(t2.Sub(t1).Nanoseconds())/1_000_000.0)

				if len(pathS) > 1 && pathS[1] == "apis" {
					l.Info().Str("type", "access").Timestamp().Msg(msg)
				} else {
					l.Debug().Str("type", "access").Timestamp().Msg(msg)
				}
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
