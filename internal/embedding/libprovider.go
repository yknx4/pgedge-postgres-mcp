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
	"fmt"
	"sync/atomic"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/ollama"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/openai"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/voyage"
)

// libProvider is the implementation of Provider backed by
// pgedge-go-llm-lib. Constructed via NewProvider; satisfies the
// existing Provider interface so tool consumers compile unchanged.
//
// Dimensions are not known up front because the library does not
// expose them ahead of an Embed call. The first successful Embed
// populates the cached dimensions atomically; Dimensions() returns
// 0 until that happens.
type libProvider struct {
	inner    llm.Client
	provider string
	model    string
	dim      atomic.Int32
}

// NewProvider constructs a Provider backed by pgedge-go-llm-lib.
// Supported provider names: "voyage", "openai", "ollama".
func NewProvider(cfg Config) (Provider, error) {
	opts, err := optionsForConfig(cfg)
	if err != nil {
		return nil, err
	}

	inner, err := llm.NewClient(cfg.Provider, opts)
	if err != nil {
		return nil, fmt.Errorf("create %s client: %w", cfg.Provider, err)
	}

	return &libProvider{
		inner:    inner,
		provider: cfg.Provider,
		model:    opts.Model,
	}, nil
}

// optionsForConfig translates our Config into llm.Options for the
// named provider, applying provider-specific defaults.
func optionsForConfig(cfg Config) (llm.Options, error) {
	switch cfg.Provider {
	case "voyage":
		if cfg.VoyageAPIKey == "" {
			return llm.Options{}, fmt.Errorf("Voyage AI API key is required when provider is 'voyage'")
		}
		model := cfg.Model
		if model == "" {
			model = "voyage-3-lite"
		}
		return llm.Options{
			APIKey:  cfg.VoyageAPIKey,
			Model:   model,
			BaseURL: cfg.VoyageBaseURL,
		}, nil

	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return llm.Options{}, fmt.Errorf("OpenAI API key is required when provider is 'openai'")
		}
		return llm.Options{
			APIKey:  cfg.OpenAIAPIKey,
			Model:   cfg.Model,
			BaseURL: cfg.OpenAIBaseURL,
		}, nil

	case "ollama":
		baseURL := cfg.OllamaURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := cfg.Model
		if model == "" {
			model = "nomic-embed-text"
		}
		return llm.Options{
			Model:   model,
			BaseURL: baseURL,
		}, nil

	default:
		return llm.Options{}, fmt.Errorf("unsupported embedding provider: %s (supported: voyage, openai, ollama)", cfg.Provider)
	}
}

// Embed generates an embedding vector and lazily caches the dimension.
func (p *libProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	vec, err := p.inner.Embed(ctx, text)
	if err != nil {
		return nil, err
	}
	if d := int32(len(vec)); d > 0 && p.dim.Load() == 0 {
		p.dim.Store(d)
	}
	return vec, nil
}

// Dimensions returns the cached embedding dimension, or 0 if Embed
// has not yet been called successfully.
func (p *libProvider) Dimensions() int {
	return int(p.dim.Load())
}

// ModelName returns the configured model name.
func (p *libProvider) ModelName() string {
	return p.model
}

// ProviderName returns the provider name ("voyage" / "openai" / "ollama").
func (p *libProvider) ProviderName() string {
	return p.provider
}
