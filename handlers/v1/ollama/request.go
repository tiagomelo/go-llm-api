// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

// generateRequest represents the request body for generating text using the Ollama API.
type generateRequest struct {
	Model   string `json:"model" validate:"required"`
	Prompt  string `json:"prompt" validate:"required"`
	Context []int  `json:"context,omitempty"`
}
