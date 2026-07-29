package http

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewServer creates and configures an HTTP server for in-cluster traffic.
func NewServer(l *zap.Logger, name, addr string, handler http.Handler, middlewares ...Middleware) *http.Server {
	// net/http's own errors belong at error level
	errLog, err := zap.NewStdLogAt(l.Named("http.server"), zapcore.ErrorLevel)
	if err != nil {
		errLog = zap.NewStdLog(l)
	}

	// Explicit HTTP/1.1. Clients reaching this server through a ClusterIP
	// Service are pinned to one pod per TCP connection by kube-proxy, so h2
	// multiplexing would concentrate their traffic on a single replica; the
	// client side is configured to match. Add proto.SetUnencryptedHTTP2(true)
	// if a gRPC-gateway or an Envoy probe needs h2c. Requires Go 1.24+.
	var proto http.Protocols
	proto.SetHTTP1(true)

	return &http.Server{
		Addr:     addr,
		Handler:  Compose(l, name, handler, middlewares...),
		ErrorLog: errLog,

		// Slowloris guard: bounds only the request line and headers, so it can
		// be strict without limiting body size or streaming.
		ReadHeaderTimeout: 5 * time.Second,

		// Absolute deadline for reading the whole body — it does not reset on
		// progress, so it effectively caps upload size by wall-clock time.
		// Handlers accepting large uploads should clear it via
		// http.NewResponseController and bound the body with
		// http.MaxBytesReader instead.
		ReadTimeout: 30 * time.Second,

		// Absolute deadline for writing the response, likewise not reset on
		// progress. Streaming handlers (SSE, watch, exports) must clear it with
		// http.NewResponseController(w).SetWriteDeadline(time.Time{}).
		WriteTimeout: 60 * time.Second,

		// Must exceed every client's IdleConnTimeout so the client is always the
		// side that closes an idle connection. Otherwise both expire together
		// and the client hits a FIN mid-request, surfacing as intermittent EOF.
		// Clients here use 30s (internal) and 60s (external).
		IdleTimeout: 120 * time.Second,

		MaxHeaderBytes: 1 << 20,
		Protocols:      &proto,
	}
}
