package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// Config holds all configuration needed by this app.
type Config struct {
	OllamaContainerName                          string `envconfig:"OLLAMA_CONTAINER_NAME" required:"true"`                              // TODO: set the correct data type.
	OllamaHost                                   string `envconfig:"OLLAMA_HOST" required:"true"`                                        // TODO: set the correct data type.
	OllamaPort                                   int    `envconfig:"OLLAMA_PORT" required:"true"`                                        // TODO: set the correct data type.
	OllamaModelName                              string `envconfig:"OLLAMA_MODEL_NAME" required:"true"`                                  // TODO: set the correct data type.
	DockerNetworkName                            string `envconfig:"DOCKER_NETWORK_NAME" required:"true"`                                // TODO: set the correct data type.
	OllamaHttpClientTimeoutSeconds               int    `envconfig:"OLLAMA_HTTP_CLIENT_TIMEOUT_SECONDS" required:"true"`                 // TODO: set the correct data type.
	OllamaHttpClientKeepAliveSeconds             int    `envconfig:"OLLAMA_HTTP_CLIENT_KEEP_ALIVE_SECONDS" required:"true"`              // TODO: set the correct data type.
	OllamaHttpClientIdleConnTimeoutSeconds       int    `envconfig:"OLLAMA_HTTP_CLIENT_IDLE_CONN_TIMEOUT_SECONDS" required:"true"`       // TODO: set the correct data type.
	OllamaHttpClientTlsHandshakeTimeoutSeconds   int    `envconfig:"OLLAMA_HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT_SECONDS" required:"true"`   // TODO: set the correct data type.
	OllamaHttpClientExpectContinueTimeoutSeconds int    `envconfig:"OLLAMA_HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT_SECONDS" required:"true"` // TODO: set the correct data type.
	GoLlmApiPort                                 int    `envconfig:"GO_LLM_API_PORT" required:"true"`                                    // TODO: set the correct data type.
}

// For ease of unit testing.
var (
	godotenvLoad     = godotenv.Load
	envconfigProcess = envconfig.Process
)

// Read reads configuration from environment variables.
// It assumes that an '.env' file is present at current path.
func Read() (*Config, error) {
	if err := godotenvLoad(); err != nil {
		return nil, errors.Wrap(err, "loading env vars from .env file")
	}
	config := new(Config)
	if err := envconfigProcess("", config); err != nil {
		return nil, errors.Wrap(err, "processing env vars")
	}
	return config, nil
}

// ReadFromEnvFile reads configuration from the specified environment file.
func ReadFromEnvFile(envFilePath string) (*Config, error) {
	if err := godotenvLoad(envFilePath); err != nil {
		return nil, errors.Wrapf(err, "loading env vars from %s", envFilePath)
	}
	config := new(Config)
	if err := envconfigProcess("", config); err != nil {
		return nil, errors.Wrap(err, "processing env vars")
	}
	return config, nil
}
