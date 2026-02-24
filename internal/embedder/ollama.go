package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient is an HTTP client for the Ollama embedding API.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Model returns the model name.
func (c *OllamaClient) Model() string {
	return c.model
}

// IsAvailable checks if Ollama is running and the model is loaded.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	// Check if our model is in the list.
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	if json.Unmarshal(body, &result) != nil {
		return false
	}

	for _, m := range result.Models {
		if m.Name == c.model || m.Name == c.model+":latest" {
			return true
		}
	}
	return false
}

// embedRequest is the Ollama /api/embed request body.
type embedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
}

// embedResponse is the Ollama /api/embed response body.
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed generates embeddings for multiple texts in a single batch.
func (c *OllamaClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	body := embedRequest{
		Model: c.model,
		Input: texts,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("embedder: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("embedder: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedder: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedder: ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("embedder: decode response: %w", err)
	}

	return result.Embeddings, nil
}

// EmbedForSearch generates an embedding for a search query.
// Adds "search_query: " prefix for models that support instruction-based embedding.
func (c *OllamaClient) EmbedForSearch(ctx context.Context, query string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{"search_query: " + query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder: no embeddings returned")
	}
	return vecs[0], nil
}

// EmbedForStorage generates an embedding for storing a document.
// Adds "search_document: " prefix for models that support instruction-based embedding.
func (c *OllamaClient) EmbedForStorage(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{"search_document: " + text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder: no embeddings returned")
	}
	return vecs[0], nil
}

// Dims returns the embedding dimensions by generating a test embedding.
func (c *OllamaClient) Dims(ctx context.Context) (int, error) {
	vecs, err := c.Embed(ctx, []string{"test"})
	if err != nil {
		return 0, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return 0, fmt.Errorf("embedder: empty embedding returned")
	}
	return len(vecs[0]), nil
}
