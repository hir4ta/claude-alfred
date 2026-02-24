package embedder

import (
	"context"
	"sync"
)

// DefaultJAModel is the default embedding model for Japanese locale.
const DefaultJAModel = "kun432/cl-nagoya-ruri-large"

// DefaultENModel is the default embedding model for other locales.
const DefaultENModel = "nomic-embed-text"

// Embedder wraps an OllamaClient with availability state.
type Embedder struct {
	client    *OllamaClient
	available bool
	dims      int
	mu        sync.RWMutex
}

// NewEmbedder creates an Embedder that gracefully degrades when Ollama is unavailable.
func NewEmbedder(baseURL, model string) *Embedder {
	return &Embedder{
		client: NewOllamaClient(baseURL, model),
	}
}

// EnsureAvailable checks Ollama availability and caches the result.
func (e *Embedder) EnsureAvailable(ctx context.Context) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.available {
		return true
	}

	if e.client.IsAvailable(ctx) {
		e.available = true
		// Cache dims.
		if dims, err := e.client.Dims(ctx); err == nil {
			e.dims = dims
		}
	}
	return e.available
}

// Available returns the cached availability status.
func (e *Embedder) Available() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.available
}

// Dims returns cached embedding dimensions.
func (e *Embedder) Dims() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dims
}

// Model returns the model name.
func (e *Embedder) Model() string {
	return e.client.Model()
}

// EmbedBatch generates embeddings for multiple texts.
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.client.Embed(ctx, texts)
}

// EmbedForSearch generates a search query embedding.
func (e *Embedder) EmbedForSearch(ctx context.Context, query string) ([]float32, error) {
	return e.client.EmbedForSearch(ctx, query)
}

// EmbedForStorage generates a document embedding.
func (e *Embedder) EmbedForStorage(ctx context.Context, text string) ([]float32, error) {
	return e.client.EmbedForStorage(ctx, text)
}

// ModelForLocale returns the recommended model for a locale code.
func ModelForLocale(localeCode string) string {
	if localeCode == "ja" {
		return DefaultJAModel
	}
	return DefaultENModel
}
