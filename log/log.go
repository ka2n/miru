package log

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/motemen/go-loghttp"
)

// Logger is the global logger instance
var Logger *slog.Logger

// InitLogger initializes the global logger
// It sets the log level to Debug if MIRU_DEBUG is set
func InitLogger() {
	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
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
