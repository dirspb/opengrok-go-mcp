package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rokasklive/opengrok-go-mcp/internal/config"
	"github.com/rokasklive/opengrok-go-mcp/internal/cursor"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

type fakeBackend struct {
	projects    []string
	projectsErr error

	searchRequests []opengrok.SearchRequest
	searchResult   opengrok.SearchResult
	searchErr      error

	fileContent   string
	fileContents  map[string]string // key: "project:filePath"
	fileErr       error
	fileErrors    map[string]error // key: "project:filePath"
	fileProject   string
	filePath      string
	fileCallCount int
	mu            sync.Mutex
}

func (b *fakeBackend) ListProjects(context.Context) ([]string, error) {
	if b.projectsErr != nil {
		return nil, b.projectsErr
	}
	return b.projects, nil
}

func (b *fakeBackend) Search(_ context.Context, req opengrok.SearchRequest) (opengrok.SearchResult, error) {
	b.searchRequests = append(b.searchRequests, req)
	if b.searchErr != nil {
		return opengrok.SearchResult{Hits: []opengrok.Hit{}}, b.searchErr
	}
	return b.searchResult, nil
}

func (b *fakeBackend) FileContent(_ context.Context, project string, filePath string) (string, error) {
	b.mu.Lock()
	b.fileProject = project
	b.filePath = filePath
	b.fileCallCount++
	b.mu.Unlock()
	key := project + ":" + filePath
	if b.fileErrors != nil {
		if err, ok := b.fileErrors[key]; ok {
			return "", err
		}
	}
	if b.fileErr != nil {
		return "", b.fileErr
	}
	if b.fileContents != nil {
		if content, ok := b.fileContents[key]; ok {
			return content, nil
		}
	}
	return b.fileContent, nil
}

func TestReadFileResourceMatchesSlashContainingPath(t *testing.T) {
	ctx := context.Background()
	backend := &fakeBackend{
		fileContent: "final class Engine {}",
	}
	server := NewMCPServer(testConfig(), backend, "test")
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect returned error: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect returned error: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "opengrok://project/platform/files/src/services/Engine.swift",
	})
	if err != nil {
		t.Fatalf("ReadResource returned error: %v", err)
	}
	if backend.fileProject != "platform" {
		t.Fatalf("FileContent project = %q, want platform", backend.fileProject)
	}
	if backend.filePath != "src/services/Engine.swift" {
		t.Fatalf("FileContent path = %q, want src/services/Engine.swift", backend.filePath)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("contents length = %d, want 1", len(result.Contents))
	}

	var output FileContextOutput
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &output); err != nil {
		t.Fatalf("resource JSON unmarshal returned error: %v", err)
	}
	if output.Content != "final class Engine {}" {
		t.Fatalf("content = %q, want file body", output.Content)
	}
}

func TestReadFileResourceLineFragmentSelectsContext(t *testing.T) {
	ctx := context.Background()
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	server := NewMCPServer(testConfig(), backend, "test")
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect returned error: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect returned error: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "opengrok://project/platform/files/src/services/Engine.swift#L2",
	})
	if err != nil {
		t.Fatalf("ReadResource returned error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("contents length = %d, want 1", len(result.Contents))
	}

	var output FileContextOutput
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &output); err != nil {
		t.Fatalf("resource JSON unmarshal returned error: %v", err)
	}
	if output.LineNumber != 2 {
		t.Fatalf("LineNumber = %d, want 2", output.LineNumber)
	}
	if output.Content != "one\ntwo\nthree" {
		t.Fatalf("Content = %q, want selected context around line 2", output.Content)
	}
}

func TestGetFileContextSlicesAroundLine(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\nfour\nfive\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:    "platform",
		FilePath:   "src/Engine.swift",
		LineNumber: 3,
		Before:     1,
		After:      1,
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.StartLine != 2 {
		t.Fatalf("StartLine = %d, want 2", output.StartLine)
	}
	if output.EndLine != 4 {
		t.Fatalf("EndLine = %d, want 4", output.EndLine)
	}
	if output.Content != "two\nthree\nfour" {
		t.Fatalf("Content = %q, want selected lines", output.Content)
	}
	if output.DisplayURL != "https://grok.example.com/source/xref/platform/src/Engine.swift#3" {
		t.Fatalf("DisplayURL = %q, want line anchor", output.DisplayURL)
	}
	if output.Citation.URL != "https://grok.example.com/source/xref/platform/src/Engine.swift#3" {
		t.Fatalf("Citation.URL = %q, want display URL", output.Citation.URL)
	}
	if output.ResourceURI != "opengrok://project/platform/files/src/Engine.swift#L3" {
		t.Fatalf("ResourceURI = %q, want line anchor", output.ResourceURI)
	}
}

func TestGetFileContextWithoutLineNumberReturnsFullFile(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", output.StartLine)
	}
	if output.EndLine != 3 {
		t.Fatalf("EndLine = %d, want 3", output.EndLine)
	}
	if output.Content != "one\ntwo\nthree\n" {
		t.Fatalf("Content = %q, want full file", output.Content)
	}
	if output.DisplayURL != "https://grok.example.com/source/xref/platform/src/Engine.swift" {
		t.Fatalf("DisplayURL = %q, want file URL without anchor", output.DisplayURL)
	}
	if output.ResourceURI != "opengrok://project/platform/files/src/Engine.swift" {
		t.Fatalf("ResourceURI = %q, want file resource without anchor", output.ResourceURI)
	}
}

func TestGetFileContextIncludeLinksFalseSuppressesBrowserLinks(t *testing.T) {
	includeLinks := false
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:      "platform",
		FilePath:     "src/Engine.swift",
		LineNumber:   2,
		IncludeLinks: &includeLinks,
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.DisplayURL != "" {
		t.Fatalf("DisplayURL = %q, want empty", output.DisplayURL)
	}
	if output.RawURL != nil {
		t.Fatalf("RawURL = %q, want nil", *output.RawURL)
	}
	if output.Citation.URL != "https://grok.example.com/source/xref/platform/src/Engine.swift#2" {
		t.Fatalf("Citation.URL = %q, want display URL even when links are suppressed", output.Citation.URL)
	}
	if output.ResourceURI != "opengrok://project/platform/files/src/Engine.swift#L2" {
		t.Fatalf("ResourceURI = %q, want resource URI", output.ResourceURI)
	}
}

func TestGetFileContextReturnsProjectRequired(t *testing.T) {
	cfg := testConfig()
	cfg.DefaultProject = ""
	service := NewService(cfg, &fakeBackend{})

	_, err := service.GetFileContext(context.Background(), FileContextInput{
		FilePath: "src/Engine.swift",
	})
	if err == nil {
		t.Fatal("GetFileContext error is nil, want PROJECT_REQUIRED")
	}
	if !IsCode(err, "PROJECT_REQUIRED") {
		t.Fatalf("GetFileContext error = %v, want PROJECT_REQUIRED", err)
	}
}

func TestGetFileContextProjectRequiredFalseStillRequiresProject(t *testing.T) {
	cfg := testConfig()
	cfg.ProjectRequired = false
	cfg.DefaultProject = ""
	service := NewService(cfg, &fakeBackend{})

	_, err := service.GetFileContext(context.Background(), FileContextInput{
		FilePath: "src/Engine.swift",
	})
	if err == nil {
		t.Fatal("GetFileContext error is nil, want PROJECT_REQUIRED")
	}
	if !IsCode(err, "PROJECT_REQUIRED") {
		t.Fatalf("GetFileContext error = %v, want PROJECT_REQUIRED", err)
	}
}

func TestGetFileContextUsesDefaultProject(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\n",
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		FilePath: "src/Engine.swift",
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if backend.fileProject != "platform" {
		t.Fatalf("backend project = %q, want platform", backend.fileProject)
	}
	if output.Project != "platform" {
		t.Fatalf("Project = %q, want platform", output.Project)
	}
}

func TestGetFileContextNegativeBeforeAfterClampToZero(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:    "platform",
		FilePath:   "src/Engine.swift",
		LineNumber: 2,
		Before:     -10,
		After:      -20,
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.StartLine != 2 {
		t.Fatalf("StartLine = %d, want 2", output.StartLine)
	}
	if output.EndLine != 2 {
		t.Fatalf("EndLine = %d, want 2", output.EndLine)
	}
	if output.Content != "two" {
		t.Fatalf("Content = %q, want selected line", output.Content)
	}
}

func TestGetFileContextLineBeyondEOFAnchorsToClampedLine(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:    "platform",
		FilePath:   "src/Engine.swift",
		LineNumber: 99,
		Before:     1,
		After:      1,
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.StartLine != 2 {
		t.Fatalf("StartLine = %d, want 2", output.StartLine)
	}
	if output.EndLine != 2 {
		t.Fatalf("EndLine = %d, want 2", output.EndLine)
	}
	if output.Content != "two" {
		t.Fatalf("Content = %q, want selected line", output.Content)
	}
	if output.LineNumber != 2 {
		t.Fatalf("LineNumber = %d, want 2", output.LineNumber)
	}
	if output.DisplayURL != "https://grok.example.com/source/xref/platform/src/Engine.swift#2" {
		t.Fatalf("DisplayURL = %q, want clamped line anchor", output.DisplayURL)
	}
	if output.ResourceURI != "opengrok://project/platform/files/src/Engine.swift#L2" {
		t.Fatalf("ResourceURI = %q, want clamped line anchor", output.ResourceURI)
	}
}

func TestSearchCodeUsesDefaultProjectAndBuildsNextCursor(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 25,
			Start:     0,
			End:       20,
			Hits: []opengrok.Hit{
				{
					Project:    "platform",
					FilePath:   "src/Engine.swift",
					LineNumber: 42,
					Snippet:    "final class Engine {}",
				},
			},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query: "Engine",
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if len(backend.searchRequests) != 1 {
		t.Fatalf("backend.Search calls = %d, want 1", len(backend.searchRequests))
	}
	gotReq := backend.searchRequests[0]
	if gotReq.Projects[0] != "platform" {
		t.Fatalf("project = %q, want platform", gotReq.Projects[0])
	}
	if gotReq.Limit != 20 {
		t.Fatalf("limit = %d, want 20", gotReq.Limit)
	}
	if output.NextCursor == nil || *output.NextCursor == "" {
		t.Fatal("NextCursor is empty, want non-empty cursor")
	}
	if len(output.Results) != 1 {
		t.Fatalf("results length = %d, want 1", len(output.Results))
	}
	if output.Results[0].DisplayURL != "https://grok.example.com/source/xref/platform/src/Engine.swift#42" {
		t.Fatalf("display URL = %q", output.Results[0].DisplayURL)
	}
	if output.Results[0].Citation.URL != "https://grok.example.com/source/xref/platform/src/Engine.swift#42" {
		t.Fatalf("Citation.URL = %q, want display URL", output.Results[0].Citation.URL)
	}
}

func TestSearchCodeTreatsBlankCursorAsFirstPage(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 1,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)
	blankCursor := "   "

	output, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query:  "Engine",
		Cursor: &blankCursor,
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if output.Diagnostics.OffsetUsed != 0 {
		t.Fatalf("OffsetUsed = %d, want first page offset", output.Diagnostics.OffsetUsed)
	}
	if backend.searchRequests[0].Offset != 0 {
		t.Fatalf("request offset = %d, want first page offset", backend.searchRequests[0].Offset)
	}
}

func TestSearchCodeRejectsUnknownConfiguredProject(t *testing.T) {
	cfg := testConfig()
	cfg.DefaultProject = "bam-bam-default"
	cfg.Projects = []string{"bam-bam-default"}
	service := NewService(cfg, &fakeBackend{})

	_, err := service.SearchCode(context.Background(), SearchCodeInput{
		Project: "bam-balam-main",
		Query:   "TaskAdaptor",
	})
	if err == nil {
		t.Fatal("SearchCode error is nil, want UNKNOWN_PROJECT")
	}
	if !IsCode(err, "UNKNOWN_PROJECT") {
		t.Fatalf("SearchCode error = %v, want UNKNOWN_PROJECT", err)
	}
	if !strings.Contains(err.Error(), "Omit project to use the default project") {
		t.Fatalf("SearchCode error = %q, want corrective guidance", err.Error())
	}
}

func TestSearchCodeUsesDefaultWhenProjectOmittedWithConfiguredProjects(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{Hits: []opengrok.Hit{}},
	}
	cfg := testConfig()
	cfg.DefaultProject = "bam-bam-default"
	cfg.Projects = []string{"bam-bam-default"}
	service := NewService(cfg, backend)

	_, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query: "TaskAdaptor",
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}
	if got := backend.searchRequests[0].Projects; len(got) != 1 || got[0] != "bam-bam-default" {
		t.Fatalf("backend projects = %#v, want default configured project", got)
	}
}

func TestSearchCodeReturnsProjectRequired(t *testing.T) {
	cfg := testConfig()
	cfg.DefaultProject = ""
	service := NewService(cfg, &fakeBackend{})

	_, err := service.SearchCode(context.Background(), SearchCodeInput{Query: "Engine"})
	if err == nil {
		t.Fatal("SearchCode error is nil, want PROJECT_REQUIRED")
	}
	if !IsCode(err, "PROJECT_REQUIRED") {
		t.Fatalf("SearchCode error = %v, want PROJECT_REQUIRED", err)
	}
}

func TestSearchCodeProjectRequiredFalseAllowsAllProjectSearch(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 1,
			Hits: []opengrok.Hit{
				{
					Project:    "platform",
					FilePath:   "src/Engine.swift",
					LineNumber: 42,
					Snippet:    "final class Engine {}",
				},
			},
		},
	}
	cfg := testConfig()
	cfg.ProjectRequired = false
	cfg.DefaultProject = ""
	service := NewService(cfg, backend)

	output, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query: "Engine",
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}
	if len(backend.searchRequests) != 1 {
		t.Fatalf("backend.Search calls = %d, want 1", len(backend.searchRequests))
	}
	if len(backend.searchRequests[0].Projects) != 0 {
		t.Fatalf("backend projects = %#v, want empty", backend.searchRequests[0].Projects)
	}
	if output.Project != "" {
		t.Fatalf("Project = %q, want empty", output.Project)
	}
	if len(output.Results) != 1 {
		t.Fatalf("results length = %d, want 1", len(output.Results))
	}
	if output.Results[0].Project != "platform" {
		t.Fatalf("result project = %q, want platform", output.Results[0].Project)
	}
}

func TestSearchCodeProjectRequiredFalseCursorKeepsAllProjectSearch(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 45,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.ProjectRequired = false
	cfg.DefaultProject = ""
	service := NewService(cfg, backend)

	firstPage, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query: "Engine",
	})
	if err != nil {
		t.Fatalf("first SearchCode returned error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first SearchCode returned nil cursor, want next cursor")
	}

	secondPage, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query:  "Engine",
		Cursor: firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("second SearchCode returned error: %v", err)
	}

	gotReq := backend.searchRequests[1]
	if len(gotReq.Projects) != 0 {
		t.Fatalf("backend projects = %#v, want empty", gotReq.Projects)
	}
	if gotReq.Offset != 20 {
		t.Fatalf("offset = %d, want 20", gotReq.Offset)
	}
	if secondPage.Project != "" {
		t.Fatalf("Project = %q, want empty", secondPage.Project)
	}
}

func TestSearchCodeExplicitProjectBeatsDefault(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{Hits: []opengrok.Hit{}},
	}
	cfg := testConfig()
	cfg.DefaultProject = "default"
	service := NewService(cfg, backend)

	_, err := service.SearchCode(context.Background(), SearchCodeInput{
		Project: "explicit",
		Query:   "Engine",
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	got := backend.searchRequests[0].Projects
	if len(got) != 1 || got[0] != "explicit" {
		t.Fatalf("projects = %#v, want [explicit]", got)
	}
}

func TestSearchCodeCursorSecondPageUsesCursorOffset(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 45,
			Start:     20,
			End:       40,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	firstPage, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query: "Engine",
	})
	if err != nil {
		t.Fatalf("first SearchCode returned error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first SearchCode returned nil cursor, want next cursor")
	}

	secondPage, err := service.SearchCode(context.Background(), SearchCodeInput{
		Query:  "Engine",
		Cursor: firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("second SearchCode returned error: %v", err)
	}

	gotReq := backend.searchRequests[1]
	if gotReq.Offset != 20 {
		t.Fatalf("offset = %d, want 20", gotReq.Offset)
	}
	if secondPage.Diagnostics.OffsetUsed != 20 {
		t.Fatalf("diagnostics offset = %d, want 20", secondPage.Diagnostics.OffsetUsed)
	}
	if secondPage.Diagnostics.OpenGrokStart != 20 {
		t.Fatalf("diagnostics start = %d, want 20", secondPage.Diagnostics.OpenGrokStart)
	}
	if secondPage.Diagnostics.OpenGrokMaxResults != 20 {
		t.Fatalf("diagnostics max results = %d, want 20", secondPage.Diagnostics.OpenGrokMaxResults)
	}
}

func TestSearchCodeCursorPageSizeIsCapped(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 100,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	cfg.PageSizeMax = 50
	service := NewService(cfg, backend)
	encodedCursor, err := cursor.Encode(cursor.State{
		Project:  "platform",
		Projects: []string{"platform"},
		Query:    "Engine",
		Mode:     "full_text",
		Offset:   20,
		PageSize: 500,
	})
	if err != nil {
		t.Fatalf("cursor Encode returned error: %v", err)
	}

	_, err = service.SearchCode(context.Background(), SearchCodeInput{
		Query:  "Engine",
		Cursor: &encodedCursor,
	})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if backend.searchRequests[0].Limit != 50 {
		t.Fatalf("limit = %d, want capped max 50", backend.searchRequests[0].Limit)
	}
}

func TestSearchCodeCursorRejectsMismatchedProjects(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 45,
			Hits:      []opengrok.Hit{},
		},
	}
	service := NewService(testConfig(), backend)

	firstPage, err := service.SearchCode(context.Background(), SearchCodeInput{
		Projects: []string{"platform", "tools"},
		Query:    "Engine",
	})
	if err != nil {
		t.Fatalf("first SearchCode returned error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first SearchCode returned nil cursor, want next cursor")
	}

	_, err = service.SearchCode(context.Background(), SearchCodeInput{
		Projects: []string{"platform", "other"},
		Query:    "Engine",
		Cursor:   firstPage.NextCursor,
	})
	if err == nil {
		t.Fatal("second SearchCode error is nil, want INVALID_CURSOR")
	}
	if !IsCode(err, "INVALID_CURSOR") {
		t.Fatalf("second SearchCode error = %v, want INVALID_CURSOR", err)
	}
}

func TestSearchSymbolDefinitionsUsesDefinitionModeAndSetsSymbolKind(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 1,
			Hits: []opengrok.Hit{
				{
					Project:    "platform",
					FilePath:   "src/Engine.swift",
					LineNumber: 42,
					Snippet:    "final class Engine {}",
				},
			},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.SearchSymbolDefinitions(context.Background(), SymbolSearchInput{
		Symbol: "Engine",
	})
	if err != nil {
		t.Fatalf("SearchSymbolDefinitions returned error: %v", err)
	}

	gotReq := backend.searchRequests[0]
	if gotReq.Mode != opengrok.ModeDefinition {
		t.Fatalf("mode = %q, want definition", gotReq.Mode)
	}
	if gotReq.Query != "Engine" {
		t.Fatalf("query = %q, want Engine", gotReq.Query)
	}
	if output.Results[0].Kind != "definition" {
		t.Fatalf("kind = %q, want definition", output.Results[0].Kind)
	}
	if output.Results[0].Symbol == nil || *output.Results[0].Symbol != "Engine" {
		t.Fatalf("symbol = %#v, want Engine", output.Results[0].Symbol)
	}
}

func TestListProjectsReturnsResourceURIs(t *testing.T) {
	backend := &fakeBackend{
		projects: []string{"platform", "tools"},
	}
	service := NewService(testConfig(), backend)

	output, err := service.ListProjects(context.Background(), ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	if len(output.Projects) != 2 {
		t.Fatalf("projects length = %d, want 2", len(output.Projects))
	}
	if output.Projects[0].ResourceURI != "opengrok://project/platform" {
		t.Fatalf("resource URI = %q, want project URI", output.Projects[0].ResourceURI)
	}
	if output.Projects[0].ProjectURL == "" {
		t.Fatal("project URL is empty, want stable search URL")
	}
}

func TestListProjectsUsesConfiguredProjects(t *testing.T) {
	backend := &fakeBackend{
		projectsErr: errors.New("forbidden"),
	}
	cfg := testConfig()
	cfg.Projects = []string{"platform", "tools"}
	service := NewService(cfg, backend)

	output, err := service.ListProjects(context.Background(), ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	got := []string{}
	for _, project := range output.Projects {
		got = append(got, project.Project)
	}
	if !slices.Equal(got, []string{"platform", "tools"}) {
		t.Fatalf("projects = %#v, want configured projects", got)
	}
}

func TestNewMCPServerRegistersOnlyEnabledTools(t *testing.T) {
	cfg := testConfig()
	cfg.Capabilities = config.Capabilities{
		SearchCode:              true,
		SearchSymbolDefinitions: true,
		SearchSymbolReferences:  true,
		GetFileContext:          true,
	}
	server := NewMCPServer(cfg, &fakeBackend{}, "test")
	clientSession, cleanup := connectMCPServer(t, server)
	defer cleanup()

	tools, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	got := []string{}
	for _, tool := range tools.Tools {
		got = append(got, tool.Name)
	}
	want := []string{"search_code", "search_symbol_definitions", "search_symbol_references", "get_file_context", "read_file"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("tools = %#v, want %#v", got, want)
	}
}

func TestNewMCPServerSearchCursorIsOptionalInToolSchema(t *testing.T) {
	cfg := testConfig()
	cfg.Capabilities = config.Capabilities{SearchSymbolDefinitions: true}
	server := NewMCPServer(cfg, &fakeBackend{}, "test")
	clientSession, cleanup := connectMCPServer(t, server)
	defer cleanup()

	tools, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(tools.Tools))
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("InputSchema type = %T, want map", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	for _, field := range required {
		if field == "cursor" || field == "include_links" {
			t.Fatalf("required fields = %#v, want cursor/include_links optional", required)
		}
	}
}

func TestNewMCPServerReturnsServer(t *testing.T) {
	server := NewMCPServer(testConfig(), &fakeBackend{}, "test")
	if server == nil {
		t.Fatal("NewMCPServer returned nil")
	}
}

func connectMCPServer(t *testing.T, server *mcp.Server) (*mcp.ClientSession, func()) {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect returned error: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Close()
		t.Fatalf("client.Connect returned error: %v", err)
	}

	return clientSession, func() {
		clientSession.Close()
		serverSession.Close()
	}
}

func TestListProjectsFirstPageReturnsPageAndTotal(t *testing.T) {
	projects := make([]string, 120)
	for i := range projects {
		projects[i] = fmt.Sprintf("project-%03d", i)
	}
	backend := &fakeBackend{projects: projects}
	service := NewService(testConfig(), backend)

	output, err := service.ListProjects(context.Background(), ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	if output.TotalProjects != 120 {
		t.Fatalf("TotalProjects = %d, want 120", output.TotalProjects)
	}
	if len(output.Projects) != 50 {
		t.Fatalf("projects count = %d, want 50 (first page)", len(output.Projects))
	}
	if output.NextCursor == nil {
		t.Fatal("NextCursor is nil, want non-nil (more pages remain)")
	}
	if output.Projects[0].Project != "project-000" {
		t.Fatalf("first project = %q, want project-000", output.Projects[0].Project)
	}
}

func TestListProjectsSecondPageViaCursorReturnsCorrectOffset(t *testing.T) {
	projects := make([]string, 120)
	for i := range projects {
		projects[i] = fmt.Sprintf("project-%03d", i)
	}
	backend := &fakeBackend{projects: projects}
	service := NewService(testConfig(), backend)

	firstPage, err := service.ListProjects(context.Background(), ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects first page error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first page NextCursor is nil")
	}

	secondPage, err := service.ListProjects(context.Background(), ListProjectsInput{
		Cursor: firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("ListProjects second page error: %v", err)
	}

	if len(secondPage.Projects) != 50 {
		t.Fatalf("second page count = %d, want 50", len(secondPage.Projects))
	}
	if secondPage.Projects[0].Project != "project-050" {
		t.Fatalf("second page first project = %q, want project-050", secondPage.Projects[0].Project)
	}
	if secondPage.NextCursor == nil {
		t.Fatal("second page NextCursor is nil, want non-nil (page 3 remains)")
	}
}

func TestListProjectsLastPageHasNoNextCursor(t *testing.T) {
	projects := make([]string, 60)
	for i := range projects {
		projects[i] = fmt.Sprintf("project-%03d", i)
	}
	backend := &fakeBackend{projects: projects}
	service := NewService(testConfig(), backend)

	firstPage, err := service.ListProjects(context.Background(), ListProjectsInput{})
	if err != nil {
		t.Fatalf("first ListProjects error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first page NextCursor is nil, expected more pages")
	}

	lastPage, err := service.ListProjects(context.Background(), ListProjectsInput{
		Cursor: firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("last ListProjects error: %v", err)
	}

	if len(lastPage.Projects) != 10 {
		t.Fatalf("last page count = %d, want 10", len(lastPage.Projects))
	}
	if lastPage.NextCursor != nil {
		t.Fatal("last page NextCursor is non-nil, want nil (no more pages)")
	}
}

func TestListProjectsInvalidCursorReturnsError(t *testing.T) {
	service := NewService(testConfig(), &fakeBackend{})
	badCursor := "not-a-valid-cursor!!!"

	_, err := service.ListProjects(context.Background(), ListProjectsInput{
		Cursor: &badCursor,
	})
	if err == nil {
		t.Fatal("ListProjects error is nil, want INVALID_CURSOR")
	}
	if !IsCode(err, "INVALID_CURSOR") {
		t.Fatalf("ListProjects error = %v, want INVALID_CURSOR", err)
	}
}

func TestGetFileContextFullFileUnderCapReturnsAllContent(t *testing.T) {
	// 3 lines — well under 500-line cap
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.TotalLines != 3 {
		t.Fatalf("TotalLines = %d, want 3", output.TotalLines)
	}
	if output.Truncated {
		t.Fatal("Truncated = true, want false for file under cap")
	}
	if output.NextCursor != nil {
		t.Fatal("NextCursor is non-nil, want nil for file under cap")
	}
	if output.Hint != nil {
		t.Fatal("Hint is non-nil, want nil for file under cap")
	}
	if output.StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", output.StartLine)
	}
	if output.EndLine != 3 {
		t.Fatalf("EndLine = %d, want 3", output.EndLine)
	}
	if output.Content != "one\ntwo\nthree\n" {
		t.Fatalf("Content = %q, want full file with trailing newline", output.Content)
	}
}

func TestGetFileContextFullFileOverCapTruncatesAndReturnsCursor(t *testing.T) {
	// Build a 600-line file
	lineSlice := make([]string, 600)
	for i := range lineSlice {
		lineSlice[i] = fmt.Sprintf("line %d", i+1)
	}
	content := strings.Join(lineSlice, "\n") + "\n"

	backend := &fakeBackend{fileContent: content}
	service := NewService(testConfig(), backend)

	output, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
	})
	if err != nil {
		t.Fatalf("GetFileContext returned error: %v", err)
	}

	if output.TotalLines != 600 {
		t.Fatalf("TotalLines = %d, want 600", output.TotalLines)
	}
	if !output.Truncated {
		t.Fatal("Truncated = false, want true for file over cap")
	}
	if output.StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", output.StartLine)
	}
	if output.EndLine != 500 {
		t.Fatalf("EndLine = %d, want 500", output.EndLine)
	}
	if output.NextCursor == nil {
		t.Fatal("NextCursor is nil, want cursor for truncated file")
	}
	if output.Hint == nil {
		t.Fatal("Hint is nil, want hint text for truncated file")
	}
	if !strings.Contains(*output.Hint, "600") {
		t.Fatalf("Hint = %q, want mention of total line count", *output.Hint)
	}
}

func TestGetFileContextNextPageViaCursorReturnsCorrectLines(t *testing.T) {
	lineSlice := make([]string, 600)
	for i := range lineSlice {
		lineSlice[i] = fmt.Sprintf("line %d", i+1)
	}
	content := strings.Join(lineSlice, "\n") + "\n"

	backend := &fakeBackend{fileContent: content}
	service := NewService(testConfig(), backend)

	firstPage, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
	})
	if err != nil {
		t.Fatalf("first GetFileContext error: %v", err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first page NextCursor is nil")
	}

	secondPage, err := service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
		Cursor:   firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("second GetFileContext error: %v", err)
	}

	if secondPage.StartLine != 501 {
		t.Fatalf("StartLine = %d, want 501", secondPage.StartLine)
	}
	if secondPage.EndLine != 600 {
		t.Fatalf("EndLine = %d, want 600", secondPage.EndLine)
	}
	if secondPage.Truncated {
		t.Fatal("Truncated = true, want false for last page")
	}
	if secondPage.NextCursor != nil {
		t.Fatal("NextCursor is non-nil, want nil for last page")
	}
}

func TestGetFileContextCursorFromDifferentFileReturnsError(t *testing.T) {
	backend := &fakeBackend{fileContent: "one\ntwo\n"}
	service := NewService(testConfig(), backend)

	// Get a valid cursor for a different file
	lineSlice := make([]string, 600)
	for i := range lineSlice {
		lineSlice[i] = fmt.Sprintf("line %d", i+1)
	}
	otherContent := strings.Join(lineSlice, "\n") + "\n"
	otherBackend := &fakeBackend{fileContent: otherContent}
	otherService := NewService(testConfig(), otherBackend)
	otherPage, err := otherService.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Other.swift",
	})
	if err != nil || otherPage.NextCursor == nil {
		t.Fatal("could not get cursor from other file")
	}

	// Use that cursor with a different file
	_, err = service.GetFileContext(context.Background(), FileContextInput{
		Project:  "platform",
		FilePath: "src/Engine.swift",
		Cursor:   otherPage.NextCursor,
	})
	if err == nil {
		t.Fatal("GetFileContext error is nil, want INVALID_CURSOR")
	}
	if !IsCode(err, "INVALID_CURSOR") {
		t.Fatalf("GetFileContext error = %v, want INVALID_CURSOR", err)
	}
}

func TestSearchCodeWarningFiresAboveThreshold(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 501,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.SearchCode(context.Background(), SearchCodeInput{Query: "Engine"})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if output.Warning == nil {
		t.Fatal("Warning is nil, want non-nil for total_hits > 500")
	}
	if !strings.Contains(*output.Warning, "501") {
		t.Fatalf("Warning = %q, want mention of hit count", *output.Warning)
	}
}

func TestSearchCodeWarningSilentBelowThreshold(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 499,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.SearchCode(context.Background(), SearchCodeInput{Query: "Engine"})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if output.Warning != nil {
		t.Fatalf("Warning = %q, want nil for total_hits <= 500", *output.Warning)
	}
}

func TestSearchCodeWarningSilentAtExactThreshold(t *testing.T) {
	backend := &fakeBackend{
		searchResult: opengrok.SearchResult{
			TotalHits: 500,
			Hits:      []opengrok.Hit{},
		},
	}
	cfg := testConfig()
	cfg.DefaultProject = "platform"
	service := NewService(cfg, backend)

	output, err := service.SearchCode(context.Background(), SearchCodeInput{Query: "Engine"})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}

	if output.Warning != nil {
		t.Fatalf("Warning = %q, want nil for total_hits exactly at threshold", *output.Warning)
	}
}

func TestExpandResultContextsAttachesWindow(t *testing.T) {
	lines := []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}
	backend := &fakeBackend{
		fileContent: strings.Join(lines, "\n"),
	}
	cfg := testConfig()
	cfg.ContextBefore = 5
	cfg.ContextAfter = 10
	service := NewService(cfg, backend)

	results := []Result{
		{Project: "platform", FilePath: "src/Foo.java", LineNumber: 6},
	}
	got := service.expandResultContexts(context.Background(), results)

	if got[0].Context == nil {
		t.Fatal("Context is nil, want non-nil")
	}
	if got[0].Context.StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", got[0].Context.StartLine)
	}
	if got[0].Context.EndLine != 10 {
		t.Fatalf("EndLine = %d, want 10", got[0].Context.EndLine)
	}
	want := strings.Join(lines, "\n")
	if got[0].Context.Content != want {
		t.Fatalf("Content = %q, want %q", got[0].Context.Content, want)
	}
}

func TestExpandResultContextsDeduplicatesFileByProject(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\nfour\nfive\n",
	}
	cfg := testConfig()
	cfg.ContextBefore = 1
	cfg.ContextAfter = 1
	service := NewService(cfg, backend)

	results := []Result{
		{Project: "platform", FilePath: "src/Foo.java", LineNumber: 2},
		{Project: "platform", FilePath: "src/Foo.java", LineNumber: 4},
	}
	got := service.expandResultContexts(context.Background(), results)

	if backend.fileCallCount != 1 {
		t.Fatalf("FileContent called %d times, want 1", backend.fileCallCount)
	}
	if got[0].Context == nil || got[1].Context == nil {
		t.Fatal("expected both results to have Context set")
	}
	if got[0].Context.StartLine != 1 {
		t.Fatalf("result[0].Context.StartLine = %d, want 1", got[0].Context.StartLine)
	}
	if got[1].Context.StartLine != 3 {
		t.Fatalf("result[1].Context.StartLine = %d, want 3", got[1].Context.StartLine)
	}
}

func TestExpandResultContextsDifferentProjectsSamePath(t *testing.T) {
	backend := &fakeBackend{
		fileContents: map[string]string{
			"alpha:src/Foo.java": "alpha-one\nalpha-two\nalpha-three\n",
			"beta:src/Foo.java":  "beta-one\nbeta-two\nbeta-three\n",
		},
	}
	cfg := testConfig()
	cfg.ContextBefore = 0
	cfg.ContextAfter = 0
	service := NewService(cfg, backend)

	results := []Result{
		{Project: "alpha", FilePath: "src/Foo.java", LineNumber: 2},
		{Project: "beta", FilePath: "src/Foo.java", LineNumber: 2},
	}
	got := service.expandResultContexts(context.Background(), results)

	if backend.fileCallCount != 2 {
		t.Fatalf("FileContent called %d times, want 2", backend.fileCallCount)
	}
	if got[0].Context == nil || got[1].Context == nil {
		t.Fatal("expected both results to have Context set")
	}
	if got[0].Context.Content != "alpha-two" {
		t.Fatalf("result[0] content = %q, want alpha-two", got[0].Context.Content)
	}
	if got[1].Context.Content != "beta-two" {
		t.Fatalf("result[1] content = %q, want beta-two", got[1].Context.Content)
	}
}

func TestExpandResultContextsFetchErrorLeavesContextNil(t *testing.T) {
	backend := &fakeBackend{
		fileContents: map[string]string{
			"platform:src/Good.java": "one\ntwo\nthree\n",
		},
		fileErrors: map[string]error{
			"platform:src/Bad.java": errors.New("not found"),
		},
	}
	cfg := testConfig()
	cfg.ContextBefore = 1
	cfg.ContextAfter = 1
	service := NewService(cfg, backend)

	results := []Result{
		{Project: "platform", FilePath: "src/Bad.java", LineNumber: 1},
		{Project: "platform", FilePath: "src/Good.java", LineNumber: 2},
	}
	got := service.expandResultContexts(context.Background(), results)

	if got[0].Context != nil {
		t.Fatalf("result[0].Context = %+v, want nil (fetch failed)", got[0].Context)
	}
	if got[1].Context == nil {
		t.Fatal("result[1].Context is nil, want non-nil")
	}
}

func TestExpandResultContextsWindowClampsAtFileBoundary(t *testing.T) {
	backend := &fakeBackend{
		fileContent: "one\ntwo\nthree\n",
	}
	cfg := testConfig()
	cfg.ContextBefore = 100
	cfg.ContextAfter = 100
	service := NewService(cfg, backend)

	results := []Result{
		{Project: "platform", FilePath: "src/Foo.java", LineNumber: 2},
	}
	got := service.expandResultContexts(context.Background(), results)

	if got[0].Context == nil {
		t.Fatal("Context is nil")
	}
	if got[0].Context.StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", got[0].Context.StartLine)
	}
	if got[0].Context.EndLine != 3 {
		t.Fatalf("EndLine = %d, want 3", got[0].Context.EndLine)
	}
}

func testConfig() config.Config {
	cfg := config.Default()
	cfg.OpenGrokWebBaseURL = "https://grok.example.com/source"
	cfg.DefaultProject = "platform"
	return cfg
}
