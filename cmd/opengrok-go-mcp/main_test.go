package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rokasklive/opengrok-go-mcp/internal/config"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

func TestHTTPServerUsesConfiguredTimeouts(t *testing.T) {
	handler := &noopHandler{}
	readTimeout := 2 * time.Second
	writeTimeout := 3 * time.Second

	server := newHTTPServer("127.0.0.1:0", handler, readTimeout, writeTimeout)

	if server.Addr != "127.0.0.1:0" {
		t.Fatalf("Addr = %q, want %q", server.Addr, "127.0.0.1:0")
	}
	if server.Handler != handler {
		t.Fatal("Handler does not match configured handler")
	}
	if server.ReadTimeout != readTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", server.ReadTimeout, readTimeout)
	}
	if server.WriteTimeout != writeTimeout {
		t.Fatalf("WriteTimeout = %v, want %v", server.WriteTimeout, writeTimeout)
	}
}

type noopHandler struct{}

func (h *noopHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func TestRunHelpReturnsNil(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	os.Args = []string{"opengrok-go-mcp", "--help"}
	if err := run(); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestDetectCapabilitiesFallsBackToSearchWhenProjectsForbidden(t *testing.T) {
	backend := &capabilityBackend{
		listProjectsErr: errors.New("unauthorized"),
		searchResults: map[opengrok.Mode]error{
			opengrok.ModeFullText:   nil,
			opengrok.ModeDefinition: nil,
			opengrok.ModeReference:  nil,
		},
		fileErr: errors.New("unauthorized"),
	}

	cfg := config.Default()
	cfg.ProbeFile = "platform/src/Engine.swift"
	caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities returned error: %v", err)
	}
	if caps.ListProjects {
		t.Fatal("ListProjects capability = true, want false")
	}
	if !caps.SearchCode || !caps.SearchSymbolDefinitions || !caps.SearchSymbolReferences {
		t.Fatalf("search capabilities = %#v, want all search enabled", caps)
	}
	if caps.GetFileContext {
		t.Fatal("GetFileContext capability = true, want false")
	}
}

func TestDetectCapabilitiesFailsWhenProjectsAndSearchFail(t *testing.T) {
	backend := &capabilityBackend{
		listProjectsErr: errors.New("unauthorized"),
		searchResults: map[opengrok.Mode]error{
			opengrok.ModeFullText:   errors.New("unauthorized"),
			opengrok.ModeDefinition: errors.New("unauthorized"),
			opengrok.ModeReference:  errors.New("unauthorized"),
		},
	}

	_, err := detectCapabilities(context.Background(), backend, config.Default(), func(string, ...any) {})
	if err == nil {
		t.Fatal("detectCapabilities error = nil, want error")
	}
}

func TestDetectCapabilitiesUsesConfiguredProjects(t *testing.T) {
	backend := &capabilityBackend{
		listProjectsErr: errors.New("unauthorized"),
		searchResults: map[opengrok.Mode]error{
			opengrok.ModeFullText: nil,
		},
	}

	cfg := config.Default()
	cfg.Projects = []string{"platform"}
	caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities returned error: %v", err)
	}
	if !caps.ListProjects {
		t.Fatal("ListProjects capability = false, want true for configured projects")
	}
	if got := backend.searchRequests[0].Projects; len(got) != 1 || got[0] != "platform" {
		t.Fatalf("search probe projects = %#v, want configured project", got)
	}
}

func TestDetectCapabilitiesEnablesFileContextWithRawFallback(t *testing.T) {
	backend := &capabilityBackend{
		searchResults: map[opengrok.Mode]error{
			opengrok.ModeFullText:   nil,
			opengrok.ModeDefinition: nil,
			opengrok.ModeReference:  nil,
		},
	}

	cfg := config.Default()
	cfg.OpenGrokWebBaseURL = "https://grok.example.com/source"
	caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities returned error: %v", err)
	}
	if !caps.GetFileContext {
		t.Fatal("GetFileContext capability = false, want true with raw fallback")
	}
}

type capabilityBackend struct {
	listProjectsErr error
	searchResults   map[opengrok.Mode]error
	searchRequests  []opengrok.SearchRequest
	fileErr         error
}

func (b *capabilityBackend) ListProjects(context.Context) ([]string, error) {
	if b.listProjectsErr != nil {
		return nil, b.listProjectsErr
	}
	return []string{"platform"}, nil
}

func (b *capabilityBackend) Search(_ context.Context, req opengrok.SearchRequest) (opengrok.SearchResult, error) {
	b.searchRequests = append(b.searchRequests, req)
	if err := b.searchResults[req.Mode]; err != nil {
		return opengrok.SearchResult{Hits: []opengrok.Hit{}}, err
	}
	return opengrok.SearchResult{Hits: []opengrok.Hit{}}, nil
}

func (b *capabilityBackend) FileContent(context.Context, string, string) (string, error) {
	if b.fileErr != nil {
		return "", b.fileErr
	}
	return "content", nil
}

func TestOpenGrokOptionsBasicAuthWinsWhenBothTokensAreConfigured(t *testing.T) {
	server := authHeaderServer(t, "Basic basic-token-value")
	defer server.Close()

	cfg := config.Config{
		OpenGrokAPIToken:       "api-token-value",
		OpenGrokBasicAuthToken: "basic-token-value",
	}
	client := opengrok.NewClient(
		server.URL+"/api/v1",
		server.Client(),
		opengrokOptions(cfg)...,
	)

	if _, err := client.ListProjects(context.Background()); err != nil {
		t.Fatalf("ListProjects() error = %v, want nil", err)
	}
}

func TestOpenGrokOptionsAPITokenOnlyUsesBearerAuth(t *testing.T) {
	server := authHeaderServer(t, "Bearer api-token-value")
	defer server.Close()

	cfg := config.Config{
		OpenGrokAPIToken: "api-token-value",
	}
	client := opengrok.NewClient(
		server.URL+"/api/v1",
		server.Client(),
		opengrokOptions(cfg)...,
	)

	if _, err := client.ListProjects(context.Background()); err != nil {
		t.Fatalf("ListProjects() error = %v, want nil", err)
	}
}

func TestOpenGrokOptionsConfiguresRawWebFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/file/content":
			http.Error(w, "forbidden", http.StatusUnauthorized)
		case "/source/raw/platform/src/Engine.swift":
			if _, err := w.Write([]byte("content")); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		default:
			t.Fatalf("path = %q, want API or raw web fallback", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := config.Config{OpenGrokWebBaseURL: server.URL + "/source"}
	client := opengrok.NewClient(
		server.URL+"/api/v1",
		server.Client(),
		opengrokOptions(cfg)...,
	)

	got, err := client.FileContent(context.Background(), "platform", "src/Engine.swift")
	if err != nil {
		t.Fatalf("FileContent() error = %v, want nil", err)
	}
	if got != "content" {
		t.Fatalf("FileContent() = %q, want fallback content", got)
	}
}

func authHeaderServer(t *testing.T, wantAuth string) *httptest.Server {
	t.Helper()

	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/projects/indexed" {
			errCh <- fmt.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/projects/indexed")
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}

		gotAuth := r.Header.Get("Authorization")
		if gotAuth != wantAuth {
			errCh <- fmt.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]string{}); err != nil {
			errCh <- fmt.Errorf("encode response: %w", err)
			return
		}
	}))

	t.Cleanup(func() {
		select {
		case err := <-errCh:
			t.Error(err)
		default:
		}
	})

	return server
}
