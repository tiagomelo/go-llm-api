// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

// Package middleware provides common HTTP middleware functions.
package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

// Logger is a middleware that logs the start and end of each HTTP request along with
// some additional information.
func Logger(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC()
		log.Info("request started",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remoteaddr", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
		log.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remoteaddr", r.RemoteAddr),
			slog.Duration("since", time.Since(start)),
		)
	})
}

// Compress is a middleware that applies gzip/deflate compression to HTTP
// responses, except when the client requests text/event-stream — gzipped SSE
// gets buffered and breaks token-by-token streaming.
func Compress(next http.Handler) http.Handler {
	compressed := handlers.CompressHandler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}
		compressed.ServeHTTP(w, r)
	})
}

// PanicRecovery is a middleware that recovers from panics in the application,
// preventing the server from crashing and logging the stack trace.
func PanicRecovery(next http.Handler) http.Handler {
	return handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(next)
}
