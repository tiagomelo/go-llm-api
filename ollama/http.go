// Copyright (c) 2026 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package ollama

import (
	"context"
	"io"
	"net"
	"net/http"
)

// requestBuilder defines an interface for building HTTP requests,
// allowing for easier testing and abstraction.
type requestBuilder interface {
	NewRequestWithContext(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error)
}

// defaultRequestBuilderProvider is the default implementation of the requestBuilder interface,
// using the standard library's http.NewRequestWithContext function.
type defaultRequestBuilderProvider struct{}

// NewRequestWithContext creates a new HTTP request with the given context, method, URL, and body.
func (p *defaultRequestBuilderProvider) NewRequestWithContext(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// requestBuilderProvider is the global variable that holds
// the current request builder implementation.
var requestBuilderProvider requestBuilder = &defaultRequestBuilderProvider{}

// httpClient defines an interface for making HTTP requests,
// allowing for easier testing and abstraction.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClientFactory defines a function type for creating new HTTP clients with a specified timeout.
type httpClientFactory func(params HTTPClientParams) httpClient

// newHTTPClient is the default implementation of the httpClientFactory.
//
// The given timeout is applied to connection setup and the response-header
// phase, NOT to the body read. Setting http.Client.Timeout would cover the
// entire request including the body, which kills long-running streaming
// responses (e.g. token-by-token generation) the moment they cross the
// configured limit. Per-request lifetime is controlled by the caller's
// context instead.
var newHTTPClient httpClientFactory = func(params HTTPClientParams) httpClient {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   params.Timeout,
				KeepAlive: params.KeepAlive,
			}).DialContext,
			ResponseHeaderTimeout: params.Timeout,
			IdleConnTimeout:       params.IdleConnTimeout,
			TLSHandshakeTimeout:   params.TLSHandshakeTimeout,
			ExpectContinueTimeout: params.ExpectContinueTimeout,
		},
	}
}

// isUnsuccessfulStatusCode checks if the given
// HTTP status code indicates an unsuccessful response.
func isUnsuccessfulStatusCode(statusCode int) bool {
	return statusCode < 200 || statusCode >= 300
}
