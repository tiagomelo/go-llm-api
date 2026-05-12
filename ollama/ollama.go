// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Client struct provides methods to interact with the Ollama API.
type Client struct {
	baseURL    string
	httpClient httpClient
}

// HTTPClientParams holds configuration parameters for creating an HTTP client.
type HTTPClientParams struct {
	Timeout               time.Duration
	KeepAlive             time.Duration
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
}

// NewClient creates and returns a new Client instance with the specified base URL and timeout.
func NewClient(baseURL string, params HTTPClientParams) Client {
	return Client{
		baseURL:    baseURL,
		httpClient: newHTTPClient(params),
	}
}

// GenerateParams represents the parameters for the Generate method of the Ollama API.
type GenerateParams struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Context []int  `json:"context,omitempty"`
}

// GenerateResponse represents the response from the Generate method of the Ollama API.
type GenerateResponse struct {
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int64  `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int64  `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// GenerateStreamChunk represents a chunk of the response
// from the GenerateStream method of the Ollama API.
type GenerateStreamChunk struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Model represents a model available in the Ollama API.
type Model struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

// Details represents the details of a model in the Ollama API.
type Details struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// Models represents a collection of models available in the Ollama API.
type Models struct {
	Items []Model `json:"models"`
}

// Models retrieves the list of models available in the Ollama API.
func (c Client) Models(ctx context.Context) (Models, error) {
	req, err := requestBuilderProvider.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/tags",
		nil,
	)
	if err != nil {
		return Models{}, errors.WithMessage(err, "failed to create request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		return Models{}, errors.WithMessage(err, "failed to list models from ollama")
	}
	defer resp.Body.Close()

	if isUnsuccessfulStatusCode(resp.StatusCode) {
		return Models{}, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var out Models
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Models{}, errors.WithMessage(err, "failed to decode response body from ollama")
	}

	return out, nil
}

// Generate sends a request to the Ollama API to generate text based on the provided model, prompt, and context.
func (c Client) Generate(ctx context.Context, model, prompt string, context ...int) (GenerateResponse, error) {
	reqBody := GenerateParams{
		Model:   model,
		Prompt:  prompt,
		Context: context,
		Stream:  false,
	}

	// json.Marshal should not fail for this struct.
	// I'm only checking for errors here because I don't want to swallow them if it does.
	b, err := json.Marshal(reqBody)
	if err != nil {
		return GenerateResponse{}, errors.WithMessage(err, "failed to marshal request body")
	}

	req, err := requestBuilderProvider.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/generate",
		bytes.NewReader(b),
	)
	if err != nil {
		return GenerateResponse{}, errors.WithMessage(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// consume body to allow connection reuse
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		return GenerateResponse{}, errors.WithMessage(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if isUnsuccessfulStatusCode(resp.StatusCode) {
		return GenerateResponse{}, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respBody GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return GenerateResponse{}, errors.WithMessage(err, "failed to decode response body")
	}

	return respBody, nil
}

// GenerateStream sends a streaming generation request to the Ollama API and
// returns two read-only channels: one for chunks as they arrive, and one for
// a terminal error (if any). Both channels are closed when the stream ends —
// successfully, on error, or via context cancellation — so callers can use a
// plain `for chunk := range out` loop and then read errCh afterwards.
//
// The errCh is buffered (capacity 1) so the goroutine can always deliver its
// terminal error without blocking, even if the consumer has already stopped
// reading from `out`.
func (c Client) GenerateStream(ctx context.Context, model, prompt string, context ...int) (<-chan GenerateStreamChunk, <-chan error) {
	out := make(chan GenerateStreamChunk)
	errCh := make(chan error, 1)

	go func() {
		// Closing both channels on exit lets the consumer's `range` loop
		// terminate and signals "no more error coming" on errCh.
		defer close(out)
		defer close(errCh)

		reqBody := GenerateParams{
			Model:   model,
			Prompt:  prompt,
			Context: context,
			Stream:  true, // Ollama returns NDJSON, one JSON object per line.
		}

		// json.Marshal should not fail for this struct; the check exists so
		// that if it ever does we surface it instead of silently swallowing.
		b, err := json.Marshal(reqBody)
		if err != nil {
			errCh <- errors.WithMessage(err, "failed to marshal request body")
			return
		}

		// Attaching ctx to the request lets `httpClient.Do` and the body reader
		// abort cleanly if the caller cancels — that's what makes Stop work.
		req, err := requestBuilderProvider.NewRequestWithContext(
			ctx,
			http.MethodPost,
			c.baseURL+"/generate",
			bytes.NewReader(b),
		)
		if err != nil {
			errCh <- errors.WithMessage(err, "failed to create request")
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// On transport error, drain and close any partial body so the
			// underlying connection can be returned to the pool instead of
			// being orphaned.
			if resp != nil && resp.Body != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			errCh <- errors.WithMessage(err, "failed to execute request")
			return
		}
		defer resp.Body.Close()

		if isUnsuccessfulStatusCode(resp.StatusCode) {
			errCh <- errors.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}

		// json.Decoder happily reads one JSON object at a time from a stream,
		// which matches Ollama's NDJSON output exactly — no manual line
		// splitting needed.
		decoder := json.NewDecoder(resp.Body)

		for {
			var chunk GenerateStreamChunk

			if err := decoder.Decode(&chunk); err != nil {
				// io.EOF is the normal end-of-stream signal; everything else
				// is a real decode/transport failure worth reporting.
				if err == io.EOF {
					return
				}
				errCh <- errors.WithMessage(err, "failed to decode stream chunk")
				return
			}

			// Send the chunk, but stay cancellable: if the consumer is gone
			// or the caller cancelled ctx, exit promptly instead of blocking
			// forever on `out <- chunk`.
			select {
			case out <- chunk:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}

			// Ollama marks the last chunk with done=true. Returning here
			// closes `out` via the deferred close and ends the consumer's
			// range loop cleanly.
			if chunk.Done {
				return
			}
		}
	}()

	return out, errCh
}

// Health checks the health of the Ollama API.
func (c Client) Health(ctx context.Context) error {
	req, err := requestBuilderProvider.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL,
		nil,
	)
	if err != nil {
		return errors.WithMessage(err, "failed to create request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// consume body to allow connection reuse.
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		return errors.WithMessage(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if isUnsuccessfulStatusCode(resp.StatusCode) {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
