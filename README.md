[![CI](https://github.com/tiagomelo/go-llm-api/actions/workflows/ci.yml/badge.svg)](https://github.com/tiagomelo/go-llm-api/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/tiagomelo/go-llm-api.svg)](https://pkg.go.dev/github.com/tiagomelo/go-llm-api)

# go-llm-api

A small Go REST API that wraps a local [Ollama](https://ollama.com) instance and exposes it through a clean, versioned HTTP interface — including a **token-by-token streaming endpoint** over Server-Sent Events.

Built on top of [`tiagomelo/go-templates/example-rest-api`](https://github.com/tiagomelo/go-templates/tree/main/example-rest-api), which provides the routing, structured logging, middleware, graceful shutdown, and Swagger plumbing.

> Walkthrough article: [*Building a streaming LLM API in Go with Ollama — and watching it run from a SwiftUI iOS app*](https://tiagomelo.info/golang/ollama/llm/streaming/ios/2026/05/11/go-llm-api-streaming.html) 

---

## Architecture

```
┌────────────────┐      HTTP / SSE       ┌──────────────┐      HTTP        ┌──────────┐
│  SwiftUI app   │  ───────────────────► │   Go API     │ ───────────────► │  Ollama  │
│  (iOS / iPad)  │  ◄─────────────────── │  (this repo) │ ◄─────────────── │ (Docker) │
└────────────────┘  data: {"response":…} └──────────────┘  ndjson chunks   └──────────┘
```

The Go API is the only thing that talks to Ollama. The iOS app talks to the Go API. Streaming flows end-to-end: as Ollama emits each token, the Go server forwards it as an SSE frame, and the iOS app appends it to the screen as it arrives.

---

## API endpoints

| Method | Path                         | Description                                            |
|--------|------------------------------|--------------------------------------------------------|
| GET    | `/api/v1/models`             | List locally available models                          |
| POST   | `/api/v1/generate`           | Non-streaming generation (single JSON response)        |
| POST   | `/api/v1/generate/stream`    | Token-by-token generation over Server-Sent Events      |

---

## Prerequisites

- [Go](https://golang.org/dl/) 1.26+
- [Docker](https://www.docker.com/) + Docker Compose (for Ollama and Swagger UI)
- For the iOS app (optional): macOS, Xcode 15+, and [`xcodegen`](https://github.com/yonaskolb/XcodeGen). See [`mobile/README.md`](./mobile/README.md).

---

## Configuration

All runtime configuration lives in `.env` at the repo root:

```env
# Ollama Docker Configuration
OLLAMA_CONTAINER_NAME=ollama
OLLAMA_HOST=localhost
OLLAMA_PORT=11434
OLLAMA_MODEL_NAME=llama3.2:1b
DOCKER_NETWORK_NAME=ollama_network

# Ollama HTTP Client Configuration
OLLAMA_HTTP_CLIENT_TIMEOUT_SECONDS=30
OLLAMA_HTTP_CLIENT_KEEP_ALIVE_SECONDS=30
OLLAMA_HTTP_CLIENT_IDLE_CONN_TIMEOUT_SECONDS=90
OLLAMA_HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT_SECONDS=10
OLLAMA_HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT_SECONDS=1

# Go LLM API Configuration
GO_LLM_API_PORT=4000
```

Note: the HTTP client timeout is applied to dial and response-header phases only — not the body — so streaming responses are not bounded by `OLLAMA_HTTP_CLIENT_TIMEOUT_SECONDS`. Per-request lifetime is controlled by the caller's `context.Context`.

---

## Quickstart

```bash
make run-ollama      # bring up Ollama in Docker
make download-model  # pull OLLAMA_MODEL_NAME inside the container
make run-api         # start the Go API on :4000
```

`make run-api` also makes sure Ollama is reachable before booting the API, so in practice you can just run that one target — it brings up Ollama if it isn't already running and pulls the model if needed.

Sanity-check the streaming endpoint with `curl`:

```bash
curl -N -s \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST http://localhost:4000/api/v1/generate/stream \
  -d '{"model":"llama3.2:1b","prompt":"say hello"}'
```

You should see frames trickle in one at a time. The `-N` flag is essential — without it curl buffers the response and you'll see everything at once.

---

## Project structure

```
.
├── cmd/                      # Application entry point
│   └── main.go
├── config/                   # Env-based config loader
├── ollama/                   # Ollama client (Models, Generate, GenerateStream)
│   ├── ollama.go
│   ├── http.go               # http.Client factory tuned for streaming
│   └── ollama_test.go
├── handlers/                 # HTTP layer
│   ├── handlers.go
│   └── v1/
│       ├── v1.go             # v1 router + middleware wiring
│       ├── v1_test.go
│       └── ollama/           # Ollama-specific HTTP handlers
│           ├── ollama.go
│           ├── request.go
│           └── ollama_test.go
├── middleware/               # Logger, Compress (SSE-aware), PanicRecovery
├── validate/                 # Request validation helpers
├── web/                      # JSON request/response helpers
├── doc/                      # Swagger annotations + generated spec
├── mobile/                   # SwiftUI iOS demo app (XcodeGen-managed)
├── docker-compose.yml        # Ollama service
├── Makefile
└── .env
```

---

## Make targets

| Target            | Description                                                    |
|-------------------|----------------------------------------------------------------|
| `help`            | Show all available targets                                     |
| `test`            | Run unit tests with the race detector                          |
| `coverage`        | Generate `coverage.html` (no `-race`, so `covermode=set` works)|
| `run-ollama`      | Start the Ollama container                                     |
| `stop-ollama`     | Stop the Ollama container                                      |
| `check-ollama`    | Verify Ollama is reachable at `${OLLAMA_HOST}:${OLLAMA_PORT}`  |
| `download-model`  | Pull `${OLLAMA_MODEL_NAME}` inside the Ollama container        |
| `run-api`         | Start the Go API on `${GO_LLM_API_PORT}`                       |
| `swagger`         | Regenerate the OpenAPI spec from code annotations              |
| `swagger-ui`      | Launch Swagger UI in Docker on port 80                         |

---

## Middleware

Applied to all v1 routes:

- **Logger** — structured JSON logging of method, path, remote address, and duration via `slog`.
- **Compress** — gzip for normal responses, **bypassed for `Accept: text/event-stream`** so SSE frames are not buffered by the compressor.
- **PanicRecovery** — recovers from panics and logs stack traces.

---

## Graceful shutdown

The server listens for `SIGINT` and `SIGTERM` and gives in-flight requests up to 5 seconds to complete before shutting down.

---

## Testing

```bash
make test       # unit tests, race detector on
make coverage   # writes coverage.html (open in a browser)
```

`make test` runs with `-race`, which forces `covermode=atomic` — that produces frequency-tinted "grey" coverage that's easy to misread as uncovered. `make coverage` runs without `-race` and pins `-covermode=set`, so the HTML report is binary red/green.

---

## Swagger documentation

```bash
make swagger      # regenerate doc/swagger.json from code annotations
make swagger-ui   # launch Swagger UI in Docker on port 80
```

Then open [http://localhost](http://localhost) for an interactive playground covering all three endpoints — including the streaming one.

---

## License

MIT — see [`LICENSE`](./LICENSE).
