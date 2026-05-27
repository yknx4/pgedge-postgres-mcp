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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLibProvider_Voyage_Embed_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "data":[{"embedding":[0.1,0.2,0.3,0.4],"index":0}],
            "model":"voyage-3-lite",
            "usage":{"total_tokens":5}
        }`))
	}))
	defer server.Close()

	p, err := NewProvider(Config{
		Provider:      "voyage",
		Model:         "voyage-3-lite",
		VoyageAPIKey:  "test-key",
		VoyageBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	vec, err := p.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 4 {
		t.Errorf("expected 4-dim vector, got %d", len(vec))
	}
	if p.Dimensions() != 4 {
		t.Errorf("Dimensions() = %d, want 4", p.Dimensions())
	}
	if p.ProviderName() != "voyage" {
		t.Errorf("ProviderName() = %q", p.ProviderName())
	}
	if p.ModelName() != "voyage-3-lite" {
		t.Errorf("ModelName() = %q", p.ModelName())
	}
}

func TestLibProvider_OpenAI_Embed_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "object":"list",
            "data":[{"object":"embedding","embedding":[0.5,0.6,0.7],"index":0}],
            "model":"text-embedding-3-small",
            "usage":{"prompt_tokens":3,"total_tokens":3}
        }`))
	}))
	defer server.Close()

	p, err := NewProvider(Config{
		Provider:      "openai",
		Model:         "text-embedding-3-small",
		OpenAIAPIKey:  "test-key",
		OpenAIBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	vec, err := p.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 {
		t.Errorf("vec len = %d", len(vec))
	}
	if p.Dimensions() != 3 {
		t.Errorf("Dimensions() = %d", p.Dimensions())
	}
}

func TestLibProvider_Ollama_Embed_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/embed") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.9,0.8]]}`))
	}))
	defer server.Close()

	p, err := NewProvider(Config{
		Provider:  "ollama",
		Model:     "nomic-embed-text",
		OllamaURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	vec, err := p.Embed(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 2 {
		t.Errorf("vec len = %d", len(vec))
	}
	if p.Dimensions() != 2 {
		t.Errorf("Dimensions() = %d, want 2", p.Dimensions())
	}
}

func TestLibProvider_Dimensions_LazyOnFirstEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1,1,1,1,1,1,1,1],"index":0}],"model":"x","usage":{"total_tokens":1}}`))
	}))
	defer server.Close()

	p, err := NewProvider(Config{
		Provider:      "voyage",
		Model:         "voyage-3-lite",
		VoyageAPIKey:  "test-key",
		VoyageBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if p.Dimensions() != 0 {
		t.Errorf("Dimensions() before Embed = %d, want 0", p.Dimensions())
	}
	if _, err := p.Embed(context.Background(), "x"); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if p.Dimensions() != 8 {
		t.Errorf("Dimensions() after Embed = %d, want 8", p.Dimensions())
	}
}

func TestLibProvider_Embed_PropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer server.Close()

	p, err := NewProvider(Config{
		Provider:      "voyage",
		Model:         "voyage-3-lite",
		VoyageAPIKey:  "wrong",
		VoyageBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	_, err = p.Embed(context.Background(), "x")
	if err == nil {
		t.Fatal("expected upstream error, got nil")
	}
}
