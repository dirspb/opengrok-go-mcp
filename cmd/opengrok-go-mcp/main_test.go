// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestDetectCapabilitiesFallsBackToDefaultProjectWhenAPIForbidden(t *testing.T) {
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
	cfg.DefaultProject = "platform"
	cfg.ProbeFile = "platform/src/Engine.swift"
	caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities returned error: %v", err)
	}
	if !caps.ListProjects {
		t.Fatal("ListProjects capability = false, want true (falls back to default project)")
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

func TestDetectCapabilitiesProbesListFiles(t *testing.T) {
	t.Run("ListFiles success enables capability", func(t *testing.T) {
		backend := &capabilityBackend{
			searchResults: map[opengrok.Mode]error{
				opengrok.ModeFullText:   nil,
				opengrok.ModeDefinition: nil,
				opengrok.ModeReference:  nil,
			},
		}

		cfg := config.Default()
		cfg.DefaultProject = "platform"
		caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
		if err != nil {
			t.Fatalf("detectCapabilities returned error: %v", err)
		}
		if !caps.ListFiles {
			t.Fatal("ListFiles capability = false, want true")
		}
	})

	t.Run("ListFiles failure disables capability", func(t *testing.T) {
		backend := &capabilityBackend{
			listFilesErr: errors.New("unauthorized"),
			searchResults: map[opengrok.Mode]error{
				opengrok.ModeFullText:   nil,
				opengrok.ModeDefinition: nil,
				opengrok.ModeReference:  nil,
			},
		}

		cfg := config.Default()
		cfg.DefaultProject = "platform"
		caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
		if err != nil {
			t.Fatalf("detectCapabilities returned error: %v", err)
		}
		if caps.ListFiles {
			t.Fatal("ListFiles capability = true, want false")
		}
	})
}

func TestDetectCapabilitiesPreservesConfiguredMemoryCapability(t *testing.T) {
	backend := &capabilityBackend{
		searchResults: map[opengrok.Mode]error{
			opengrok.ModeFullText:   nil,
			opengrok.ModeDefinition: nil,
			opengrok.ModeReference:  nil,
		},
	}

	cfg := config.Default()
	cfg.DefaultProject = "platform"
	caps, err := detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities returned error: %v", err)
	}
	if !caps.Memory {
		t.Fatal("Memory capability = false, want configured local capability preserved")
	}

	cfg.Capabilities.Memory = false
	caps, err = detectCapabilities(context.Background(), backend, cfg, func(string, ...any) {})
	if err != nil {
		t.Fatalf("detectCapabilities disabled-memory error: %v", err)
	}
	if caps.Memory {
		t.Fatal("Memory capability = true, want configured disable preserved")
	}
}

type capabilityBackend struct {
	listProjectsErr error
	listFilesErr    error
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

func (b *capabilityBackend) ListFiles(context.Context, string, string) ([]opengrok.FileEntry, error) {
	if b.listFilesErr != nil {
		return nil, b.listFilesErr
	}
	return []opengrok.FileEntry{}, nil
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

func TestStartupLogsDerivedConfig(t *testing.T) {
	t.Setenv("OPENGROK_MCP_WEB_BASE_URL", "")
	t.Setenv("OPENGROK_MCP_DEFAULT_PROJECT", "")
	t.Setenv("OPENGROK_MCP_PROJECTS", "")

	cfg := config.Default()
	cfg.OpenGrokAPIBaseURL = "https://grok.example.com/source/api/v1"
	cfg.OpenGrokWebBaseURL = "https://grok.example.com/source"
	cfg.Projects = []string{"platform"}
	cfg.DefaultProject = "platform"
	cfg.Capabilities = config.Capabilities{
		ListProjects: true,
		SearchCode:   true,
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	logStartupDiagnostics(cfg, "")

	output := buf.String()
	if !strings.Contains(output, "web URL") && !strings.Contains(output, "derived") {
		t.Fatalf("expected log output to contain 'web URL' or 'derived', got: %s", output)
	}
}

func TestStartupLogsEnabledCapabilities(t *testing.T) {
	cfg := config.Default()
	cfg.OpenGrokAPIBaseURL = "https://grok.example.com/source/api/v1"
	cfg.OpenGrokWebBaseURL = "https://grok.example.com/source"
	cfg.DefaultProject = "platform"
	cfg.Capabilities = config.Capabilities{
		ListProjects:            true,
		SearchCode:              false,
		SearchSymbolDefinitions: true,
		SearchSymbolReferences:  false,
		GetFileContext:          true,
		ListSymbols:             false,
		ListFiles:               true,
		Memory:                  true,
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	logStartupDiagnostics(cfg, "")

	output := buf.String()
	for _, name := range []string{
		"list_projects", "search_code", "search_symbol_definitions",
		"search_symbol_references", "get_file_context", "list_symbols", "list_files", "memory",
	} {
		if !strings.Contains(output, name) {
			t.Fatalf("expected capability name %q in log output, got: %s", name, output)
		}
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
