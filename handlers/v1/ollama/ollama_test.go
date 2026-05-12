// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tiagomelo/go-llm-api/ollama"
)

func TestModels(t *testing.T) {
	t.Run("should return models successfully", func(t *testing.T) {
		mockClient := mockOllamaClient{
			listModels: func(ctx context.Context) (ollama.Models, error) {
				return ollama.Models{
					Items: []ollama.Model{
						{Name: "model1", Model: "model1"},
						{Name: "model2", Model: "model2"},
					},
				}, nil
			},
		}

		req, err := http.NewRequest(http.MethodGet, "api/v1/models", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockClient)
		handler := http.HandlerFunc((h).Models)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), "model1")
		require.Contains(t, rr.Body.String(), "model2")
	})

	t.Run("should handle client error", func(t *testing.T) {
		mockClient := mockOllamaClient{
			listModels: func(ctx context.Context) (ollama.Models, error) {
				return ollama.Models{}, errors.New("test error")
			},
		}

		req, err := http.NewRequest(http.MethodGet, "api/v1/models", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockClient)
		handler := http.HandlerFunc((h).Models)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		require.Contains(t, rr.Body.String(), "test error")
	})
}

func TestGenerate(t *testing.T) {
	t.Run("should generate response successfully", func(t *testing.T) {
		mockClient := mockOllamaClient{
			generate: func(ctx context.Context, model, prompt string, context ...int) (ollama.GenerateResponse, error) {
				return ollama.GenerateResponse{
					Response: "generated response",
				}, nil
			},
		}

		payload := `{"model":"test-model","prompt":"test prompt"}`
		req, err := http.NewRequest(http.MethodPost, "api/v1/generate", bytes.NewBuffer([]byte(payload)))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockClient)
		handler := http.HandlerFunc((h).Generate)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), "generated response")
	})

	t.Run("invalid json should return bad request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "api/v1/generate", bytes.NewBuffer([]byte(`{`)))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		handler := http.HandlerFunc((h).Generate)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing model should return bad request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "api/v1/generate", bytes.NewBuffer([]byte(`{"prompt":"test prompt"}`)))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		handler := http.HandlerFunc((h).Generate)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "model is a required field")
	})

	t.Run("missing prompt should return bad request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "api/v1/generate", bytes.NewBuffer([]byte(`{"model":"test-model"}`)))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		handler := http.HandlerFunc((h).Generate)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "prompt is a required field")
	})

	t.Run("should handle client error", func(t *testing.T) {
		mockClient := mockOllamaClient{
			generate: func(ctx context.Context, model, prompt string, context ...int) (ollama.GenerateResponse, error) {
				return ollama.GenerateResponse{}, errors.New("test error")
			},
		}

		payload := `{"model":"test-model","prompt":"test prompt"}`
		req, err := http.NewRequest(http.MethodPost, "api/v1/generate", bytes.NewBuffer([]byte(payload)))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h := New(mockClient)
		handler := http.HandlerFunc((h).Generate)
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		require.Contains(t, rr.Body.String(), "test error")
	})
}

func TestGenerateStream(t *testing.T) {
	t.Run("should stream chunks successfully", func(t *testing.T) {
		chunks := make(chan ollama.GenerateStreamChunk)
		errs := make(chan error, 1)

		mockClient := mockOllamaClient{
			generateStream: func(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error) {
				go func() {
					defer close(chunks)
					defer close(errs)

					chunks <- ollama.GenerateStreamChunk{Response: "Hello", Done: false}
					chunks <- ollama.GenerateStreamChunk{Response: " world", Done: false}
					chunks <- ollama.GenerateStreamChunk{Done: true}
				}()
				return chunks, errs
			},
		}

		payload := `{"model":"test-model","prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockClient)
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
		require.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
		require.Equal(t, "keep-alive", rr.Header().Get("Connection"))
		require.Equal(t, "no", rr.Header().Get("X-Accel-Buffering"))

		body := rr.Body.String()
		require.Contains(t, body, `data: {"response":"Hello","done":false}`)
		require.Contains(t, body, `data: {"response":" world","done":false}`)
		require.Contains(t, body, `event: done`)
		require.Contains(t, body, `data: {"done":true}`)
	})

	t.Run("invalid json should return bad request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(`{`))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing model should return bad request", func(t *testing.T) {
		payload := `{"prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "model is a required field")
	})

	t.Run("missing prompt should return bad request", func(t *testing.T) {
		payload := `{"model":"test-model"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockOllamaClient{})
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "prompt is a required field")
	})

	t.Run("should return error when flusher is not supported", func(t *testing.T) {
		mockClient := mockOllamaClient{}

		payload := `{"model":"test-model","prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		w := &noFlusherResponseWriter{
			header: make(http.Header),
		}

		h := New(mockClient)
		http.HandlerFunc(h.GenerateStream).ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.status)
		require.Contains(t, w.body.String(), "streaming not supported")
	})

	t.Run("should handle stream error", func(t *testing.T) {
		chunks := make(chan ollama.GenerateStreamChunk)
		errs := make(chan error, 1)

		mockClient := mockOllamaClient{
			generateStream: func(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error) {
				go func() {
					defer close(chunks)
					defer close(errs)

					errs <- errors.New("stream error")
				}()
				return chunks, errs
			},
		}

		payload := `{"model":"test-model","prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockClient)
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)

		body := rr.Body.String()
		require.Contains(t, body, `event: error`)
		require.Contains(t, body, `stream error`)
	})

	t.Run("closed chunks channel should return without body events", func(t *testing.T) {
		chunks := make(chan ollama.GenerateStreamChunk)
		errs := make(chan error, 1)

		mockClient := mockOllamaClient{
			generateStream: func(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error) {
				close(chunks)
				close(errs)
				return chunks, errs
			},
		}

		payload := `{"model":"test-model","prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h := New(mockClient)
		http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		require.NotContains(t, rr.Body.String(), `data:`)
		require.NotContains(t, rr.Body.String(), `event:`)
	})

	t.Run("should stop when context is canceled", func(t *testing.T) {
		chunks := make(chan ollama.GenerateStreamChunk)
		errs := make(chan error, 1)

		mockClient := mockOllamaClient{
			generateStream: func(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error) {
				go func() {
					defer close(chunks)
					defer close(errs)

					// keep sending so handler blocks on select
					for {
						chunks <- ollama.GenerateStreamChunk{Response: "x", Done: false}
					}
				}()
				return chunks, errs
			},
		}

		payload := `{"model":"test-model","prompt":"hi"}`
		req, err := http.NewRequest(http.MethodPost, "/api/v1/generate/stream", bytes.NewBufferString(payload))
		require.NoError(t, err)

		// 👇 create cancellable context
		ctx, cancel := context.WithCancel(req.Context())
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		h := New(mockClient)

		// run handler async
		done := make(chan struct{})
		go func() {
			http.HandlerFunc(h.GenerateStream).ServeHTTP(rr, req)
			close(done)
		}()

		// give it time to enter loop
		time.Sleep(10 * time.Millisecond)

		// 👇 trigger this branch
		cancel()

		select {
		case <-done:
			// success: handler exited
		case <-time.After(1 * time.Second):
			t.Fatal("handler did not stop after context cancel")
		}
	})
}

type mockOllamaClient struct {
	listModels     func(ctx context.Context) (ollama.Models, error)
	generate       func(ctx context.Context, model, prompt string, context ...int) (ollama.GenerateResponse, error)
	generateStream func(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error)
}

func (m mockOllamaClient) Models(ctx context.Context) (ollama.Models, error) {
	return m.listModels(ctx)
}

func (m mockOllamaClient) Generate(ctx context.Context, model, prompt string, context ...int) (ollama.GenerateResponse, error) {
	return m.generate(ctx, model, prompt, context...)
}

func (m mockOllamaClient) GenerateStream(ctx context.Context, model, prompt string, context ...int) (<-chan ollama.GenerateStreamChunk, <-chan error) {
	return m.generateStream(ctx, model, prompt, context...)
}

type noFlusherResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *noFlusherResponseWriter) Header() http.Header {
	return w.header
}

func (w *noFlusherResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *noFlusherResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}
