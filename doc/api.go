// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package doc

// swagger:route GET /api/v1/models models GetModels
// Get all available models.
// ---
// responses:
//   200: getModelsResponse
//   500: description: internal server error

// swagger:response getModelsResponse
type GetModelsResponseWrapper struct {
	// in:body
	Body GetModelsResponse
}

type GetModelsResponse struct {
	Models []Model `json:"models"`
}

// swagger:route POST /api/v1/generate generate Generate
// Generate text based on the provided model, prompt, and context.
// ---
// responses:
//   200: generateResponse
//   400: description: bad request
//   500: description: internal server error

// swagger:parameters Generate
type GenerateRequestWrapper struct {
	// in:body
	// required: true
	Body GenerateRequest
}

// swagger:response generateResponse
type GenerateResponseWrapper struct {
	// in:body
	Body GenerateResponse
}

type GenerateRequest struct {
	// required: true
	// example: llama3.2:1b
	Model string `json:"model"`

	// required: true
	// example: hi
	Prompt string `json:"prompt"`

	// example: [128006,9125]
	Context []int `json:"context,omitempty"`
}

type GenerateResponse struct {
	// example: How can I help you today?
	Response string `json:"response"`

	// example: true
	Done bool `json:"done"`

	// example: stop
	DoneReason string `json:"done_reason"`

	// example: [128006,9125,128007,271]
	Context []int `json:"context"`

	// example: 2220439376
	TotalDuration int64 `json:"total_duration"`

	// example: 1300497293
	LoadDuration int64 `json:"load_duration"`

	// example: 26
	PromptEvalCount int64 `json:"prompt_eval_count"`

	// example: 455812958
	PromptEvalDuration int64 `json:"prompt_eval_duration"`

	// example: 8
	EvalCount int64 `json:"eval_count"`

	// example: 459100459
	EvalDuration int64 `json:"eval_duration"`
}

// swagger:route POST /api/v1/generate/stream generate GenerateStream
// Stream generated text using Server-Sent Events (SSE).
//
// Each event is formatted as:
//
//	`data: {"response":"token","done":false}`
//
// Final event:
//
//	`event: done`
//	`data: {"done":true}`
//
// ---
// consumes:
// - application/json
// produces:
// - text/event-stream
// responses:
//   200: description: SSE stream of generated tokens
//   400: description: bad request
//   500: description: internal server error

// swagger:parameters GenerateStream
type GenerateStreamRequestWrapper struct {
	// in:body
	// required: true
	Body GenerateRequest
}

// swagger:model StreamChunk
type StreamChunk struct {
	// example: How
	Response string `json:"response"`

	// example: false
	Done bool `json:"done"`
}

type Model struct {
	// example: llama3.2:1b
	Name string `json:"name"`

	// example: llama3.2:1b
	Model string `json:"model"`

	// example: 2026-04-28T11:38:38.445479013Z
	ModifiedAt string `json:"modified_at"`

	// example: 1321098329
	Size int64 `json:"size"`

	// example: baf6a787fdffd633537aa2eb51cfd54cb93ff08e28040095462bb63daf552878
	Digest string `json:"digest"`

	Details Details `json:"details"`
}

type Details struct {
	// example: gguf
	Format string `json:"format"`

	// example: llama
	Family string `json:"family"`

	// example: ["llama"]
	Families []string `json:"families"`

	// example: 1.2B
	ParameterSize string `json:"parameter_size"`

	// example: Q8_0
	QuantizationLevel string `json:"quantization_level"`
}
