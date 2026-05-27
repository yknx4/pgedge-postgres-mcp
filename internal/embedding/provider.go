/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package embedding

import (
	"context"
)

// Provider defines the interface for embedding generation
type Provider interface {
	// Embed generates an embedding vector for the given text
	Embed(ctx context.Context, text string) ([]float64, error)

	// Dimensions returns the number of dimensions in the embedding vector
	Dimensions() int

	// ModelName returns the name of the model being used
	ModelName() string

	// ProviderName returns the name of the provider (e.g., "voyage", "ollama", "openai")
	ProviderName() string
}

// Config holds configuration for embedding providers
type Config struct {
	Provider string // "voyage", "ollama", or "openai"
	Model    string // Model name (provider-specific)

	// Voyage AI-specific
	VoyageAPIKey  string
	VoyageBaseURL string // Base URL for Voyage API (optional, uses default if empty)

	// OpenAI-specific
	OpenAIAPIKey  string
	OpenAIBaseURL string // Base URL for OpenAI API (optional, uses default if empty)

	// Ollama-specific
	OllamaURL string
}
