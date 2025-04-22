package log

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/motemen/go-loghttp"
)

// Logger is the global logger instance
var Logger *slog.Logger

const (
	ADD_SOURCE = false
)

// InitLogger initializes the global logger
// It sets the log level to Debug if MIRU_DEBUG is set
func InitLogger() {
	opts := &slog.HandlerOptions{
		AddSource: ADD_SOURCE,
		Level:     slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if ADD_SOURCE {
				// skip the call stack for the logger itself
				if a.Key == slog.SourceKey {
					const skip = 7
					_, file, line, ok := runtime.Caller(skip)
					if !ok {
						return a
					}
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(file), line))
				}
			}

			return a
		},
	}

	if os.Getenv("MIRU_DEBUG") != "" {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	loghttp.DefaultTransport.LogRequest = func(req *http.Request) {
		Debug("HTTP request",
			"method", req.Method,
			"url", req.URL.String(),
			"headers", req.Header,
		)
	}

	loghttp.DefaultTransport.LogResponse = func(resp *http.Response) {
		Debug("HTTP response",
			"method", resp.Request.Method,
			"url", resp.Request.URL.String(),
			"status", resp.Status,
			"status_code", resp.StatusCode,
			"headers", resp.Header,
		)
	}
}

// init initializes the logger when the package is imported
func init() {
	InitLogger()
}

func EnableGlobalHTTP() {
	http.DefaultTransport = loghttp.DefaultTransport
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}
