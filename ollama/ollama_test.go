// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModels(t *testing.T) {
	originalRequestBuilderProvider := requestBuilderProvider
	originalNewHTTPClient := newHTTPClient

	t.Run("should return a list of models", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
			requestBuilderProvider = originalRequestBuilderProvider
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			models := `
			{
				"models": [
					{
					"name": "llama3.2:1b",
					"model": "llama3.2:1b",
					"modified_at": "2026-04-28T11:38:38.445479013Z",
					"size": 1321098329,
					"digest": "baf6a787fdffd633537aa2eb51cfd54cb93ff08e28040095462bb63daf552878",
					"details": {
						"format": "gguf",
						"family": "llama",
						"families": [
						"llama"
						],
						"parameter_size": "1.2B",
						"quantization_level": "Q8_0"
					}
					}
				]
			}
			`
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(models)),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		expectedModels := Models{
			Items: []Model{
				{
					Name:       "llama3.2:1b",
					Model:      "llama3.2:1b",
					ModifiedAt: time.Date(2026, 4, 28, 11, 38, 38, 445479013, time.UTC),
					Size:       1321098329,
					Digest:     "baf6a787fdffd633537aa2eb51cfd54cb93ff08e28040095462bb63daf552878",
					Details: Details{
						Format:            "gguf",
						Family:            "llama",
						Families:          []string{"llama"},
						ParameterSize:     "1.2B",
						QuantizationLevel: "Q8_0",
					},
				},
			},
		}

		models, err := client.Models(context.TODO())
		require.NoError(t, err)

		require.Equal(t, expectedModels, models)
	})

	t.Run("should return an error if the request builder fails", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
		}()

		requestBuilderProvider = &mockRequestBuilderProvider{
			req: nil,
			err: io.ErrUnexpectedEOF,
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Models(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("should return an error if the HTTP client fails", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: nil,
				err:  io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Models(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to list models from ollama: unexpected EOF")
	})

	t.Run("should return an error if the HTTP client fails and discard body", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("some response")),
				},
				err: io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Models(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to list models from ollama: unexpected EOF")
	})

	t.Run("should return an error if the response status code is not 200", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				},
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Models(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("should return an error if the response body cannot be decoded", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Models(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode response body")
	})
}

func TestGenerate(t *testing.T) {
	originalRequestBuilderProvider := requestBuilderProvider
	originalNewHTTPClient := newHTTPClient

	t.Run("should generate a response from the model", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			resp := `
			{
				"response": "How can I help you today?",
				"done": true,
				"done_reason": "stop",
				"context": [
					128006,
					9125
				],
				"total_duration": 2220439376,
				"load_duration": 1300497293,
				"prompt_eval_count": 26,
				"prompt_eval_duration": 455812958,
				"eval_count": 8,
				"eval_duration": 459100459
			}
			`
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		expectedResponse := GenerateResponse{
			Response:           "How can I help you today?",
			Done:               true,
			DoneReason:         "stop",
			Context:            []int{128006, 9125},
			TotalDuration:      2220439376,
			LoadDuration:       1300497293,
			PromptEvalCount:    26,
			PromptEvalDuration: 455812958,
			EvalCount:          8,
			EvalDuration:       459100459,
		}

		response, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.NoError(t, err)

		require.Equal(t, expectedResponse, response)
	})

	t.Run("should return an error if the request builder fails", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
		}()

		requestBuilderProvider = &mockRequestBuilderProvider{
			req: nil,
			err: io.ErrUnexpectedEOF,
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("should return an error if the HTTP client fails", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: nil,
				err:  io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the HTTP client fails and discard response body", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("some content")),
				},
				err: io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the response status code is not 200", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				},
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("should return an error if the response body cannot be decoded", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		_, err := client.Generate(context.TODO(), "llama3.2:1b", "Hello, how are you?")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode response body")
	})
}

func TestGenerateStream(t *testing.T) {
	originalRequestBuilderProvider := requestBuilderProvider
	originalNewHTTPClient := newHTTPClient

	t.Run("should stream the generated response from the model", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
			newHTTPClient = originalNewHTTPClient
		}()

		streamData := `{"response":"Hello, ","done":false}` + "\n" +
			`{"response":"how are you?","done":true}`

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(streamData)),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello, how are you?")

		var chunks []GenerateStreamChunk
		for chunk := range out {
			chunks = append(chunks, chunk)
		}

		require.NoError(t, <-errCh)
		require.Equal(t, []GenerateStreamChunk{
			{Response: "Hello, ", Done: false},
			{Response: "how are you?", Done: true},
		}, chunks)
	})

	t.Run("should return an error if the request builder fails", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
		}()

		requestBuilderProvider = &mockRequestBuilderProvider{
			req: nil,
			err: io.ErrUnexpectedEOF,
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello")
		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("should return an error if the HTTP client fails", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: nil,
				err:  io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello")
		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the HTTP client fails and discard response body", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("some content")),
				},
				err: io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello")
		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the response status code is not 200", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				},
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello")
		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("should return an error if a stream chunk cannot be decoded", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		out, errCh := client.GenerateStream(context.TODO(), "llama3.2:1b", "Hello")
		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode stream chunk")
	})

	t.Run("should return ctx.Err() if the context is canceled mid-stream", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		streamData := `{"response":"Hello, ","done":false}` + "\n" +
			`{"response":"how are you?","done":true}`

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(streamData)),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})
		ctx, cancel := context.WithCancel(context.Background())

		out, errCh := client.GenerateStream(ctx, "llama3.2:1b", "Hello")

		first, ok := <-out
		require.True(t, ok)
		require.Equal(t, "Hello, ", first.Response)

		cancel()

		for range out {
		}

		err := <-errCh
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestHealth(t *testing.T) {
	originalRequestBuilderProvider := requestBuilderProvider
	originalNewHTTPClient := newHTTPClient

	t.Run("should return no error if API is responding fine", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			resp := `Ollama is running`

			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
				},
				err: nil,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		err := client.Health(context.TODO())
		require.NoError(t, err)
	})

	t.Run("should return an error if the request builder fails", func(t *testing.T) {
		defer func() {
			requestBuilderProvider = originalRequestBuilderProvider
		}()

		requestBuilderProvider = &mockRequestBuilderProvider{
			req: nil,
			err: io.ErrUnexpectedEOF,
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		err := client.Health(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("should return an error if the HTTP client fails", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: nil,
				err:  io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		err := client.Health(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the HTTP client fails and discard body", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("some content")),
				},
				err: io.ErrUnexpectedEOF,
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		err := client.Health(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute request: unexpected EOF")
	})

	t.Run("should return an error if the response status code is not 200", func(t *testing.T) {
		defer func() {
			newHTTPClient = originalNewHTTPClient
		}()

		newHTTPClient = func(_ HTTPClientParams) httpClient {
			return &mockHTTPClient{
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				},
			}
		}

		client := NewClient("http://localhost:11434", HTTPClientParams{})

		err := client.Health(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected status code: 500")
	})
}

type mockRequestBuilderProvider struct {
	req *http.Request
	err error
}

func (m *mockRequestBuilderProvider) NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return m.req, m.err
}

type mockHTTPClient struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.resp, m.err
}
