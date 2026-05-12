// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tiagomelo/go-llm-api/ollama"
	"github.com/tiagomelo/go-llm-api/validate"
	"github.com/tiagomelo/go-llm-api/web"
)

// jsonDecode decodes a JSON request body into a given struct.
var jsonDecode = func(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// handlers struct holds any dependencies for the handlers.
type handlers struct {
	client ollamaClient
}

// ollamaClient defines the methods we need from the Ollama client.
type ollamaClient interface {
	// Models retrieves the list of available models from the Ollama API.
	Models(ctx context.Context) (ollama.Models, error)

	// Generate sends a prompt to the Ollama API and returns the generated response.
	Generate(ctx context.Context, model, prompt string, context ...int) (ollama.GenerateResponse, error)

	// GenerateStream sends a prompt to the Ollama API and returns a channel
	// for streaming the generated response chunks.
	GenerateStream(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error)
}

// New initializes a new instance of handlers.
func New(client ollamaClient) *handlers {
	return &handlers{
		client: client,
	}
}

func (h handlers) Models(w http.ResponseWriter, r *http.Request) {
	models, err := h.client.Models(r.Context())
	if err != nil {
		web.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.RespondWithJson(w, http.StatusOK, models)
}

func (h handlers) Generate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var generateReq generateRequest
	if err := jsonDecode(r.Body, &generateReq); err != nil {
		web.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validate.Check(generateReq); err != nil {
		web.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.client.Generate(r.Context(), generateReq.Model, generateReq.Prompt, generateReq.Context...)
	if err != nil {
		web.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.RespondWithJson(w, http.StatusOK, resp)
}

func (h handlers) GenerateStream(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var generateReq generateRequest
	if err := jsonDecode(r.Body, &generateReq); err != nil {
		web.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Check(generateReq); err != nil {
		web.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		web.RespondWithError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	chunks, errs := h.client.GenerateStream(
		r.Context(),
		generateReq.Model,
		generateReq.Prompt,
		generateReq.Context...,
	)

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return
			}

			if chunk.Done {
				fmt.Fprintf(w, "event: done\ndata: {\"done\":true}\n\n")
				flusher.Flush()
				return
			}

			// json.Marshal should not fail for this struct (only string/bool fields).
			// This is a defensive check to avoid breaking the SSE stream on unexpected changes.
			b, err := json.Marshal(chunk)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"failed to marshal stream chunk\"}\n\n")
				flusher.Flush()
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()

		case err, ok := <-errs:
			if ok && err != nil {
				b, _ := json.Marshal(map[string]string{
					"error": err.Error(),
				})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", b)
				flusher.Flush()
				return
			}

		case <-r.Context().Done():
			return
		}
	}
}
