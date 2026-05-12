include .env
export

# ==============================================================================
# Help

.PHONY: help
## help: shows this help message
help:
	@ echo "Usage: make [target]\n"
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==============================================================================
# Tests

.PHONY: test
## test: run unit tests with race detector
test:
	@ go test -v -race ./... -count=1

.PHONY: coverage
## coverage: generate coverage report (no -race, so covermode=set is honored)
coverage:
	@set -e; \
	packages=$$(go list ./... | grep -v "/cmd"); \
	if [ -z "$$packages" ]; then \
		echo "No valid Go packages found"; \
		exit 1; \
	fi; \
	echo "Packages:" $$packages; \
	go test -covermode=set -count=1 -coverpkg=$$(echo $$packages | tr ' ' ',') -coverprofile=coverage.out $$packages; \
	go tool cover -html=coverage.out -o coverage.html; \
	echo "Generated: coverage.out coverage.html"

# ==============================================================================
# Swagger

.PHONY: swagger
## swagger: generates api's documentation
swagger: 
	@ unset `env|grep DOCKER|cut -d\= -f1` ;\
	docker run --rm --name books-swagger -it -v $(HOME):$(HOME) -w $(PWD) quay.io/goswagger/swagger generate spec -o doc/swagger.json

.PHONY: swagger-ui
## swagger-ui: launches swagger ui
swagger-ui: swagger
	@ docker run --rm --name books-swagger-ui -p 80:8080 -e SWAGGER_JSON=/docs/swagger.json -v $(shell pwd)/doc:/docs swaggerapi/swagger-ui

# ==============================================================================
# Docker

.PHONY: run-ollama
## run-ollama: runs ollama in a docker container
run-ollama:
	@ docker-compose up -d ${OLLAMA_CONTAINER_NAME}

.PHONY: stop-ollama
## stop-ollama: stops ollama container
stop-ollama:
	@ docker-compose down ${OLLAMA_CONTAINER_NAME}

.PHONY: check-ollama
## check-ollama: checks if ollama is reachable
check-ollama:
	@ if ! curl -fsS --max-time 3 -o /dev/null http://${OLLAMA_HOST}:${OLLAMA_PORT}; then echo >&2 "Ollama is not reachable at http://${OLLAMA_HOST}:${OLLAMA_PORT}. Please run 'make run-ollama' first."; exit 1; fi

# ==============================================================================
# App's execution

.PHONY: run-api
## run-api: runs the API
run-api: run-ollama check-ollama
	@ go run cmd/main.go -p ${GO_LLM_API_PORT}

# ==============================================================================
# Ollama

.PHONY: download-model
## download-model: downloads the model to be used by ollama
download-model: run-ollama check-ollama
	@ docker exec -it ${OLLAMA_CONTAINER_NAME} ollama pull ${OLLAMA_MODEL_NAME}