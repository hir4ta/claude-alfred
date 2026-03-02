package llm

import (
	"fmt"
	"os"
)

// Client wraps the Anthropic API for LLM-based extraction tasks.
// ALFRED_API_KEY is required; NewClient returns an error if unset.
type Client struct {
	client *anthropicClient
}

// NewClient creates a Client that reads the API key from ALFRED_API_KEY.
func NewClient() (*Client, error) {
	apiKey := os.Getenv("ALFRED_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ALFRED_API_KEY is required but not set")
	}
	return &Client{
		client: newAnthropicClient(apiKey),
	}, nil
}
