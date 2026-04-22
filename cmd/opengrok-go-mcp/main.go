package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rokasklive/opengrok-go-mcp/internal/config"
	"github.com/rokasklive/opengrok-go-mcp/internal/mcpserver"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		log.Printf("opengrok-go-mcp: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.FromEnv()

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	if err := cfg.RegisterFlags(fs); err != nil {
		return fmt.Errorf("register flags: %w", err)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.ReadTimeout,
	}
	backend := opengrok.NewClient(
		cfg.OpenGrokAPIBaseURL,
		httpClient,
		opengrokOptions(cfg)...,
	)
	checkCtx, cancel := context.WithTimeout(context.Background(), cfg.ReadTimeout)
	defer cancel()
	caps, err := detectCapabilities(checkCtx, backend, cfg, log.Printf)
	if err != nil {
		return err
	}
	cfg.Capabilities = caps

	mcpServer := mcpserver.NewMCPServer(cfg, backend, version)
	if cfg.Transport == config.TransportStdio {
		return mcpServer.Run(context.Background(), &mcp.StdioTransport{})
	}

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	server := newHTTPServer(cfg.Listen, mux, cfg.ReadTimeout, cfg.WriteTimeout)
	return serve(server)
}

func newHTTPServer(
	addr string,
	handler http.Handler,
	readTimeout time.Duration,
	writeTimeout time.Duration,
) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

func serve(server *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err, ok := <-errCh:
		if !ok {
			return nil
		}
		return err
	case <-ctx.Done():
		stop()
	}

	shutdownTimeout := server.WriteTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err, ok := <-errCh; ok {
		return err
	}
	return nil
}

type capabilityChecker interface {
	ListProjects(context.Context) ([]string, error)
	Search(context.Context, opengrok.SearchRequest) (opengrok.SearchResult, error)
	FileContent(context.Context, string, string) (string, error)
}

func detectCapabilities(
	ctx context.Context,
	backend capabilityChecker,
	cfg config.Config,
	logf func(string, ...any),
) (config.Capabilities, error) {
	var caps config.Capabilities
	caps.ListProjects = true
	if _, err := backend.ListProjects(ctx); err != nil {
		if len(cfg.Projects) > 0 {
			logCapability(logf, "list_projects", true, "API unavailable, using configured projects")
		} else {
			logCapability(logf, "list_projects", true, "API unavailable, falling back to default project")
		}
	} else {
		logCapability(logf, "list_projects", true, "")
	}

	probeProjects := capabilityProbeProjects(cfg)
	caps.SearchCode = probeSearchCapability(
		ctx,
		backend,
		opengrok.ModeFullText,
		"test",
		probeProjects,
		logf,
		"search_code",
	)
	caps.SearchSymbolDefinitions = probeSearchCapability(
		ctx,
		backend,
		opengrok.ModeDefinition,
		"test",
		probeProjects,
		logf,
		"search_symbol_definitions",
	)
	caps.SearchSymbolReferences = probeSearchCapability(
		ctx,
		backend,
		opengrok.ModeReference,
		"test",
		probeProjects,
		logf,
		"search_symbol_references",
	)
	caps.GetFileContext = probeFileCapability(ctx, backend, cfg, logf)

	if !caps.SearchCode && !caps.SearchSymbolDefinitions && !caps.SearchSymbolReferences {
		return caps, errors.New("check OpenGrok access: no search capabilities are available")
	}
	return caps, nil
}

func capabilityProbeProjects(cfg config.Config) []string {
	switch {
	case len(cfg.Projects) > 0:
		return []string{cfg.Projects[0]}
	case cfg.DefaultProject != "":
		return []string{cfg.DefaultProject}
	default:
		return []string{}
	}
}

func probeSearchCapability(
	ctx context.Context,
	backend capabilityChecker,
	mode opengrok.Mode,
	query string,
	projects []string,
	logf func(string, ...any),
	name string,
) bool {
	_, err := backend.Search(ctx, opengrok.SearchRequest{
		Projects: projects,
		Query:    query,
		Mode:     mode,
		Limit:    1,
		Offset:   0,
	})
	if err != nil {
		logCapability(logf, name, false, err.Error())
		return false
	}

	logCapability(logf, name, true, "")
	return true
}

func probeFileCapability(
	ctx context.Context,
	backend capabilityChecker,
	cfg config.Config,
	logf func(string, ...any),
) bool {
	probeFile := cfg.ProbeFile
	if probeFile == "" {
		if cfg.OpenGrokWebBaseURL != "" {
			logCapability(logf, "get_file_context", true, "raw web fallback configured without probe file")
			return true
		}
		logCapability(logf, "get_file_context", false, "OPENGROK_MCP_PROBE_FILE and OPENGROK_MCP_WEB_BASE_URL are not configured")
		return false
	}

	project, filePath, ok := strings.Cut(strings.Trim(probeFile, "/"), "/")
	if !ok || project == "" || filePath == "" {
		logCapability(logf, "get_file_context", false, "OPENGROK_MCP_PROBE_FILE must be project/path")
		return false
	}
	if _, err := backend.FileContent(ctx, project, filePath); err != nil {
		logCapability(logf, "get_file_context", false, err.Error())
		return false
	}

	logCapability(logf, "get_file_context", true, "")
	return true
}

func logCapability(logf func(string, ...any), name string, enabled bool, reason string) {
	if reason == "" {
		logf("opengrok capability %s: enabled", name)
		return
	}
	if enabled {
		logf("opengrok capability %s: enabled: %s", name, reason)
		return
	}
	logf("opengrok capability %s: disabled: %s", name, reason)
}

func opengrokOptions(cfg config.Config) []opengrok.Option {
	options := []opengrok.Option{}
	if cfg.DefaultProject != "" {
		options = append(options, opengrok.WithDefaultProject(cfg.DefaultProject))
	}
	if cfg.OpenGrokAPIToken != "" {
		options = append(options, opengrok.WithAPIToken(cfg.OpenGrokAPIToken))
	}
	if cfg.OpenGrokBasicAuthToken != "" {
		options = append(options, opengrok.WithBasicAuthToken(cfg.OpenGrokBasicAuthToken))
	}
	if cfg.OpenGrokWebBaseURL != "" {
		options = append(options, opengrok.WithWebBaseURL(cfg.OpenGrokWebBaseURL))
	}
	if cfg.Debug {
		options = append(options, opengrok.WithDebug(true))
	}

	return options
}
