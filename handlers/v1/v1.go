// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package v1

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	ollamaV1 "github.com/tiagomelo/go-llm-api/handlers/v1/ollama"
	"github.com/tiagomelo/go-llm-api/middleware"
	"github.com/tiagomelo/go-llm-api/ollama"
)

// Config holds the configuration for the API routes, including the logger.
type Config struct {
	Log          *slog.Logger
	OllamaClient ollama.Client
}

// Routes sets up the API routes for the application and applies middleware.
func Routes(c Config) *mux.Router {
	router := mux.NewRouter()
	initializeRoutes(c, router)
	router.Use(
		func(h http.Handler) http.Handler {
			return middleware.Logger(c.Log, h)
		},
		middleware.Compress,
		middleware.PanicRecovery,
	)
	return router
}

// initializeRoutes sets up the API routes for the application.
func initializeRoutes(c Config, router *mux.Router) {
	ollamaHandlers := ollamaV1.New(c.OllamaClient)
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/v1/models", ollamaHandlers.Models).Methods(http.MethodGet)
	apiRouter.HandleFunc("/v1/generate", ollamaHandlers.Generate).Methods(http.MethodPost)
	apiRouter.HandleFunc("/v1/generate/stream", ollamaHandlers.GenerateStream).Methods(http.MethodPost)
}
