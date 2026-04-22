package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rokasklive/opengrok-go-mcp/internal/config"
	"github.com/rokasklive/opengrok-go-mcp/internal/cursor"
	"github.com/rokasklive/opengrok-go-mcp/internal/links"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

const (
	codeInvalidCursor   = "INVALID_CURSOR"
	codeProjectRequired = "PROJECT_REQUIRED"
	codeUnknownProject  = "UNKNOWN_PROJECT"

	defaultSearchMode = string(opengrok.ModeFullText)
	defaultBefore     = 30
	defaultAfter      = 60

	filePageSize        = 500
	projectsPageSize    = 50
	searchWarnThreshold = 500
)

type Backend interface {
	ListProjects(ctx context.Context) ([]string, error)
	Search(ctx context.Context, req opengrok.SearchRequest) (opengrok.SearchResult, error)
	FileContent(ctx context.Context, project string, filePath string) (string, error)
}

type Service struct {
	cfg     config.Config
	backend Backend
	links   links.Builder
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func IsCode(err error, code string) bool {
	var serviceErr *Error
	if errors.As(err, &serviceErr) {
		return serviceErr.Code == code
	}

	return false
}

func NewService(cfg config.Config, backend Backend) *Service {
	return &Service{
		cfg:     cfg,
		backend: backend,
		links:   links.NewBuilder(cfg.OpenGrokWebBaseURL, cfg.EnableRawLinks),
	}
}

func (s *Service) ListProjects(ctx context.Context, input ListProjectsInput) (ListProjectsOutput, error) {
	offset := 0
	pageSize := projectsPageSize

	if input.Cursor != nil && *input.Cursor != "" {
		state, err := cursor.DecodeProjects(*input.Cursor)
		if err != nil {
			return ListProjectsOutput{}, invalidCursorError()
		}
		offset = state.Offset
		pageSize = state.PageSize
	}

	allProjects, err := s.backend.ListProjects(ctx)
	if err != nil {
		if len(s.cfg.Projects) > 0 {
			allProjects = s.cfg.Projects
		} else {
			allProjects = []string{s.cfg.DefaultProject}
		}
	}

	total := len(allProjects)
	if offset > total {
		offset = total
	}
	end := min(offset+pageSize, total)
	page := allProjects[offset:end]

	items := make([]ProjectItem, 0, len(page))
	for _, project := range page {
		items = append(items, ProjectItem{
			Project:     project,
			Title:       project,
			Description: "Indexed OpenGrok project",
			ProjectURL:  s.links.Search(project, defaultSearchMode, ""),
			ResourceURI: s.links.Project(project),
		})
	}

	var nextCursor *string
	if offset+pageSize < total {
		encoded, err := cursor.EncodeProjects(cursor.ProjectsState{
			Offset:   offset + pageSize,
			PageSize: pageSize,
		})
		if err != nil {
			return ListProjectsOutput{}, fmt.Errorf("projects cursor: %w", err)
		}
		nextCursor = &encoded
	}

	return ListProjectsOutput{
		Projects:      items,
		TotalProjects: total,
		NextCursor:    nextCursor,
	}, nil
}

func (s *Service) SearchCode(ctx context.Context, input SearchCodeInput) (SearchOutput, error) {
	mode := input.Mode
	if mode == "" {
		mode = defaultSearchMode
	}

	return s.search(ctx, searchRequest{
		project:      input.Project,
		projects:     input.Projects,
		query:        input.Query,
		mode:         mode,
		pathPrefix:   input.PathPrefix,
		fileType:     input.FileType,
		pageSize:     input.PageSize,
		cursor:       cursorValue(input.Cursor),
		includeLinks: input.IncludeLinks,
	})
}

func (s *Service) SearchSymbolDefinitions(ctx context.Context, input SymbolSearchInput) (SearchOutput, error) {
	return s.search(ctx, searchRequest{
		project:      input.Project,
		projects:     input.Projects,
		query:        input.Symbol,
		mode:         string(opengrok.ModeDefinition),
		pageSize:     input.PageSize,
		cursor:       cursorValue(input.Cursor),
		includeLinks: input.IncludeLinks,
		symbol:       input.Symbol,
	})
}

func (s *Service) SearchSymbolReferences(ctx context.Context, input SymbolSearchInput) (SearchOutput, error) {
	return s.search(ctx, searchRequest{
		project:      input.Project,
		projects:     input.Projects,
		query:        input.Symbol,
		mode:         string(opengrok.ModeReference),
		pageSize:     input.PageSize,
		cursor:       cursorValue(input.Cursor),
		includeLinks: input.IncludeLinks,
		symbol:       input.Symbol,
	})
}

func (s *Service) GetFileContext(ctx context.Context, input FileContextInput) (FileContextOutput, error) {
	projects, err := s.resolveProjects(input.Project, nil)
	if err != nil {
		return FileContextOutput{}, err
	}

	content, err := s.backend.FileContent(ctx, projects[0], input.FilePath)
	if err != nil {
		return FileContextOutput{}, fmt.Errorf("file context: %w", err)
	}

	if input.LineNumber > 0 {
		return s.windowedFileContext(projects[0], content, input)
	}
	return s.pagedFileContext(projects[0], content, input)
}

func (s *Service) windowedFileContext(project string, content string, input FileContextInput) (FileContextOutput, error) {
	selectedContent, selectedLine, startLine, endLine := fileContextLines(content, input)
	lines := fileLines(content)
	totalLines := len(lines)
	fileLinks := s.links.File(project, input.FilePath, selectedLine)
	output := FileContextOutput{
		Project:              project,
		FilePath:             input.FilePath,
		LineNumber:           selectedLine,
		StartLine:            startLine,
		EndLine:              endLine,
		TotalLines:           totalLines,
		Truncated:            false,
		Content:              selectedContent,
		DisplayURL:           fileLinks.DisplayURL,
		RawURL:               fileLinks.RawURL,
		Citation:             citation(displayTitle(input.FilePath, selectedLine), fileLinks.DisplayURL, selectedLine),
		AnnotationsAvailable: input.IncludeAnnotations,
		ResourceURI:          fileLinks.ResourceURI,
	}
	if !s.includeLinks(input.IncludeLinks) {
		output.DisplayURL = ""
		output.RawURL = nil
	}
	return output, nil
}

func (s *Service) pagedFileContext(project string, content string, input FileContextInput) (FileContextOutput, error) {
	startLine := 1

	if input.Cursor != nil && *input.Cursor != "" {
		state, err := cursor.DecodeFile(*input.Cursor)
		if err != nil {
			return FileContextOutput{}, invalidCursorError()
		}
		if state.Project != project || state.FilePath != input.FilePath {
			return FileContextOutput{}, invalidCursorError()
		}
		startLine = state.StartLine
	}

	lines := fileLines(content)
	totalLines := len(lines)

	if totalLines == 0 {
		return FileContextOutput{
			Project:    project,
			FilePath:   input.FilePath,
			TotalLines: 0,
			Content:    "",
			ResourceURI: s.links.File(project, input.FilePath, 0).ResourceURI,
		}, nil
	}

	if startLine > totalLines {
		startLine = totalLines
	}

	endLine := min(startLine+filePageSize-1, totalLines)

	var selectedContent string
	if startLine == 1 && endLine == totalLines {
		selectedContent = content
	} else {
		selectedContent = strings.Join(lines[startLine-1:endLine], "\n")
	}

	truncated := endLine < totalLines

	fileLinks := s.links.File(project, input.FilePath, 0)

	var nextCursor *string
	var hint *string
	if truncated {
		encoded, err := cursor.EncodeFile(cursor.FileState{
			Project:   project,
			FilePath:  input.FilePath,
			StartLine: endLine + 1,
			PageSize:  filePageSize,
		})
		if err != nil {
			return FileContextOutput{}, fmt.Errorf("file cursor: %w", err)
		}
		nextCursor = &encoded
		hintText := fmt.Sprintf("File has %d lines, showing %d–%d. Pass next_cursor to read the next section.", totalLines, startLine, endLine)
		hint = &hintText
	}

	output := FileContextOutput{
		Project:              project,
		FilePath:             input.FilePath,
		LineNumber:           0,
		StartLine:            startLine,
		EndLine:              endLine,
		TotalLines:           totalLines,
		Truncated:            truncated,
		Content:              selectedContent,
		DisplayURL:           fileLinks.DisplayURL,
		RawURL:               fileLinks.RawURL,
		Citation:             citation(displayTitle(input.FilePath, 0), fileLinks.DisplayURL, 0),
		NextCursor:           nextCursor,
		Hint:                 hint,
		AnnotationsAvailable: input.IncludeAnnotations,
		ResourceURI:          fileLinks.ResourceURI,
	}
	if !s.includeLinks(input.IncludeLinks) {
		output.DisplayURL = ""
		output.RawURL = nil
	}
	return output, nil
}

func NewMCPServer(cfg config.Config, backend Backend, version string) *mcp.Server {
	service := NewService(cfg, backend)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "opengrok-go-mcp",
		Version: version,
	}, nil)

	if cfg.Capabilities.ListProjects {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "list_projects",
			Description: "List indexed OpenGrok projects.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, input ListProjectsInput) (*mcp.CallToolResult, ListProjectsOutput, error) {
			output, err := service.ListProjects(ctx, input)
			return nil, output, err
		})
		server.AddResource(&mcp.Resource{
			URI:         "opengrok://projects",
			Name:        "projects",
			Title:       "OpenGrok projects",
			Description: "Indexed OpenGrok projects.",
			MIMEType:    "application/json",
		}, service.projectsResource)
		server.AddResourceTemplate(&mcp.ResourceTemplate{
			URITemplate: "opengrok://project/{project}",
			Name:        "project",
			Title:       "OpenGrok project",
			Description: "OpenGrok project metadata.",
			MIMEType:    "application/json",
		}, service.projectResource)
	}
	if cfg.Capabilities.SearchCode {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "search_code",
			Description: "Search reference/base code in OpenGrok. Omit project unless the user explicitly names an OpenGrok project; do not infer project from the local repository name. Use mode full_text, path, history, definition, or reference. For file-name searches use mode=path. Use returned file_path/project with read_file instead of fetching display_url/raw_url yourself. When answering about a specific file or class, include the selected result's citation.url.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchCodeInput) (*mcp.CallToolResult, SearchOutput, error) {
			output, err := service.SearchCode(ctx, input)
			return nil, output, err
		})
	}
	if cfg.Capabilities.SearchSymbolDefinitions {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "search_symbol_definitions",
			Description: "Search symbol definitions in reference/base OpenGrok code. Omit project unless the user explicitly names an OpenGrok project; do not infer project from the local repository name. Use returned file_path/project with read_file to read the matched file; do not use WebFetch for display_url/raw_url because browser URLs may require auth. When answering about a class/interface, include citation.url for the definition.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, input SymbolSearchInput) (*mcp.CallToolResult, SearchOutput, error) {
			output, err := service.SearchSymbolDefinitions(ctx, input)
			return nil, output, err
		})
	}
	if cfg.Capabilities.SearchSymbolReferences {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "search_symbol_references",
			Description: "Search symbol references in reference/base OpenGrok code. Omit project unless the user explicitly names an OpenGrok project; do not infer project from the local repository name. Use returned file_path/project with read_file to read the matched file; avoid calling this for broad symbols unless you need many references. If discussing a specific reference, include citation.url.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, input SymbolSearchInput) (*mcp.CallToolResult, SearchOutput, error) {
			output, err := service.SearchSymbolReferences(ctx, input)
			return nil, output, err
		})
	}
	if cfg.Capabilities.GetFileContext {
		readFile := func(ctx context.Context, req *mcp.CallToolRequest, input FileContextInput) (*mcp.CallToolResult, FileContextOutput, error) {
			output, err := service.GetFileContext(ctx, input)
			return nil, output, err
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "get_file_context",
			Description: "Read a line window around a specific line in an OpenGrok file. Use when you have a line_number from search results and need surrounding context. Omit project unless the user explicitly names an OpenGrok project; do not infer project from the local repository name. For full-file reads use read_file instead. When answering the user about this file, include citation.url.",
		}, readFile)
		mcp.AddTool(server, &mcp.Tool{
			Name:        "read_file",
			Description: "Read full file content from OpenGrok. Use project and file_path from search results; otherwise omit project unless the user explicitly names an OpenGrok project. Do not infer project from the local repository name. Do not use WebFetch on display_url/raw_url; this tool sends configured auth and falls back to /raw. For a targeted line window use get_file_context with line_number. When summarizing a class or file, include citation.url in the final answer.",
		}, readFile)
		server.AddResourceTemplate(&mcp.ResourceTemplate{
			URITemplate: "opengrok://project/{project}/files/{+path}",
			Name:        "file",
			Title:       "OpenGrok file",
			Description: "OpenGrok file contents.",
			MIMEType:    "application/json",
		}, service.fileResource)
	}

	return server
}

func cursorValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

type searchRequest struct {
	project      string
	projects     []string
	query        string
	mode         string
	pathPrefix   string
	fileType     string
	pageSize     int
	cursor       string
	includeLinks *bool
	symbol       string
}

func (s *Service) search(ctx context.Context, req searchRequest) (SearchOutput, error) {
	projects, err := s.resolveSearchProjects(req.project, req.projects)
	if err != nil {
		return emptySearchOutput(req.mode, req.query), err
	}
	project := firstProject(projects)

	pageSize := s.pageSize(req.pageSize)
	offset := 0
	if req.cursor != "" {
		state, err := cursor.Decode(req.cursor)
		if err != nil {
			return emptySearchOutput(req.mode, req.query), invalidCursorError()
		}

		expected := cursor.State{
			Project:    project,
			Projects:   projects,
			Query:      req.query,
			Mode:       req.mode,
			PathPrefix: req.pathPrefix,
			FileType:   req.fileType,
		}
		if err := state.Validate(expected); err != nil {
			return emptySearchOutput(req.mode, req.query), invalidCursorError()
		}

		offset = state.Offset
		pageSize = s.pageSize(state.PageSize)
	}

	result, err := s.backend.Search(ctx, opengrok.SearchRequest{
		Projects:   projects,
		Query:      req.query,
		Mode:       opengrok.Mode(req.mode),
		PathPrefix: req.pathPrefix,
		FileType:   req.fileType,
		Limit:      pageSize,
		Offset:     offset,
	})
	if err != nil {
		return emptySearchOutput(req.mode, req.query), fmt.Errorf("search: %w", err)
	}

	nextCursor, err := s.nextCursor(cursor.State{
		Project:    project,
		Projects:   projects,
		Query:      req.query,
		Mode:       req.mode,
		Offset:     offset + pageSize,
		PageSize:   pageSize,
		PathPrefix: req.pathPrefix,
		FileType:   req.fileType,
	}, result.TotalHits)
	if err != nil {
		return emptySearchOutput(req.mode, req.query), fmt.Errorf("search cursor: %w", err)
	}

	return SearchOutput{
		Project:    project,
		Mode:       req.mode,
		Query:      req.query,
		TotalHits:  result.TotalHits,
		Results:    s.results(result.Hits, project, req.mode, req.symbol, s.includeLinks(req.includeLinks)),
		PageSize:   pageSize,
		NextCursor: nextCursor,
		Diagnostics: Diagnostics{
			OffsetUsed:         offset,
			OpenGrokStart:      result.Start,
			OpenGrokMaxResults: pageSize,
		},
	}, nil
}

func (s *Service) resolveSearchProjects(project string, projects []string) ([]string, error) {
	resolved, err := s.resolveProjects(project, projects)
	if err == nil {
		return resolved, nil
	}
	if s.cfg.ProjectRequired {
		return nil, err
	}

	return []string{}, nil
}

func (s *Service) resolveProjects(project string, projects []string) ([]string, error) {
	switch {
	case project != "":
		if err := s.validateConfiguredProjects([]string{project}); err != nil {
			return nil, err
		}
		return []string{project}, nil
	case len(projects) > 0:
		resolved := make([]string, len(projects))
		copy(resolved, projects)
		if err := s.validateConfiguredProjects(resolved); err != nil {
			return nil, err
		}
		return resolved, nil
	case s.cfg.DefaultProject != "":
		return []string{s.cfg.DefaultProject}, nil
	default:
		return nil, &Error{
			Code:    codeProjectRequired,
			Message: "No project selected. Pass project or call list_projects first.",
		}
	}
}

func (s *Service) validateConfiguredProjects(projects []string) error {
	if len(s.cfg.Projects) == 0 {
		return nil
	}

	allowed := map[string]bool{}
	for _, project := range s.cfg.Projects {
		allowed[project] = true
	}
	for _, project := range projects {
		if allowed[project] {
			continue
		}
		return &Error{
			Code: codeUnknownProject,
			Message: fmt.Sprintf(
				"Unknown OpenGrok project %q. Configured OpenGrok projects: %s. Omit project to use the default project %q.",
				project,
				strings.Join(s.cfg.Projects, ", "),
				s.cfg.DefaultProject,
			),
		}
	}

	return nil
}

func firstProject(projects []string) string {
	if len(projects) == 0 {
		return ""
	}

	return projects[0]
}

func (s *Service) pageSize(requested int) int {
	size := requested
	if size == 0 {
		size = s.cfg.PageSizeDefault
	}
	if s.cfg.PageSizeMax > 0 && size > s.cfg.PageSizeMax {
		size = s.cfg.PageSizeMax
	}
	if size <= 0 {
		return 20
	}

	return size
}

func (s *Service) includeLinks(value *bool) bool {
	if value != nil {
		return *value
	}

	return s.cfg.IncludeLinksDefault
}

func (s *Service) nextCursor(state cursor.State, totalHits int) (*string, error) {
	if state.Offset >= totalHits {
		return nil, nil
	}

	value, err := cursor.Encode(state)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func (s *Service) results(
	hits []opengrok.Hit,
	defaultProject string,
	mode string,
	symbol string,
	includeLinks bool,
) []Result {
	results := make([]Result, 0, len(hits))
	for _, hit := range hits {
		project := hit.Project
		if project == "" {
			project = defaultProject
		}
		fileLinks := s.links.File(project, hit.FilePath, hit.LineNumber)

		var resultSymbol *string
		if symbol != "" {
			value := symbol
			resultSymbol = &value
		}

		result := Result{
			ResultID:     project + ":" + hit.FilePath + ":" + strconv.Itoa(hit.LineNumber),
			Project:      project,
			FilePath:     hit.FilePath,
			LineNumber:   hit.LineNumber,
			ColumnNumber: nil,
			Kind:         mode,
			Symbol:       resultSymbol,
			Snippet:      hit.Snippet,
			DisplayTitle: displayTitle(hit.FilePath, hit.LineNumber),
			Citation:     citation(displayTitle(hit.FilePath, hit.LineNumber), fileLinks.DisplayURL, hit.LineNumber),
			ResourceURI:  fileLinks.ResourceURI,
			Score:        nil,
			Metadata:     map[string]any{},
		}
		if includeLinks {
			result.DisplayURL = fileLinks.DisplayURL
			result.RawURL = fileLinks.RawURL
		}

		results = append(results, result)
	}

	return results
}

func displayTitle(filePath string, lineNumber int) string {
	return path.Base(filePath) + ":" + strconv.Itoa(lineNumber)
}

func citation(title string, url string, line int) Citation {
	return Citation{
		Title: title,
		URL:   url,
		Line:  line,
	}
}

func fileContextLines(content string, input FileContextInput) (string, int, int, int) {
	lines := fileLines(content)
	totalLines := len(lines)
	if input.LineNumber <= 0 {
		if totalLines == 0 {
			return content, 0, 0, 0
		}

		return content, 0, 1, totalLines
	}
	if totalLines == 0 {
		return "", 0, 0, 0
	}

	before := contextWindow(input.Before, defaultBefore)
	after := contextWindow(input.After, defaultAfter)
	selectedLine := min(input.LineNumber, totalLines)
	startLine := max(1, input.LineNumber-before)
	endLine := min(totalLines, input.LineNumber+after)
	if startLine > totalLines {
		startLine = totalLines
	}
	if endLine < startLine {
		endLine = startLine
	}

	return strings.Join(lines[startLine-1:endLine], "\n"), selectedLine, startLine, endLine
}

func contextWindow(value int, defaultValue int) int {
	if value < 0 {
		return 0
	}
	if value == 0 {
		return defaultValue
	}

	return value
}

func fileLines(content string) []string {
	if content == "" {
		return []string{}
	}

	lines := strings.Split(content, "\n")
	if strings.HasSuffix(content, "\n") {
		return lines[:len(lines)-1]
	}

	return lines
}

func emptySearchOutput(mode string, query string) SearchOutput {
	return SearchOutput{
		Mode:    mode,
		Query:   query,
		Results: []Result{},
	}
}

func invalidCursorError() error {
	return &Error{
		Code:    codeInvalidCursor,
		Message: "Invalid cursor.",
	}
}

func (s *Service) projectsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	output, err := s.ListProjects(ctx, ListProjectsInput{})
	if err != nil {
		return nil, err
	}

	return jsonResource(req.Params.URI, output)
}

func (s *Service) projectResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	project, _, _, ok := parseProjectResourceURI(req.Params.URI)
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	output := ProjectItem{
		Project:     project,
		Title:       project,
		Description: "Indexed OpenGrok project",
		ProjectURL:  s.links.Search(project, defaultSearchMode, ""),
		ResourceURI: s.links.Project(project),
	}

	return jsonResource(req.Params.URI, output)
}

func (s *Service) fileResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	project, filePath, lineNumber, ok := parseProjectResourceURI(req.Params.URI)
	if !ok || filePath == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	output, err := s.GetFileContext(ctx, FileContextInput{
		Project:    project,
		FilePath:   filePath,
		LineNumber: lineNumber,
	})
	if err != nil {
		return nil, err
	}

	return jsonResource(req.Params.URI, output)
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func parseProjectResourceURI(rawURI string) (string, string, int, bool) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "opengrok" || parsed.Host != "project" {
		return "", "", 0, false
	}

	lineNumber, ok := parseLineFragment(parsed.Fragment)
	if !ok {
		return "", "", 0, false
	}

	rest := strings.TrimPrefix(parsed.EscapedPath(), "/")
	projectPart, filePart, hasFile := strings.Cut(rest, "/files/")
	project, err := url.PathUnescape(projectPart)
	if err != nil || project == "" {
		return "", "", 0, false
	}
	if !hasFile {
		return project, "", lineNumber, true
	}

	filePath, err := url.PathUnescape(filePart)
	if err != nil || filePath == "" {
		return "", "", 0, false
	}

	return project, filePath, lineNumber, true
}

func parseLineFragment(fragment string) (int, bool) {
	if fragment == "" {
		return 0, true
	}

	var value string
	switch {
	case strings.HasPrefix(fragment, "L"):
		value = strings.TrimPrefix(fragment, "L")
	case strings.HasPrefix(fragment, "line="):
		value = strings.TrimPrefix(fragment, "line=")
	default:
		return 0, true
	}

	lineNumber, err := strconv.Atoi(value)
	if err != nil || lineNumber <= 0 {
		return 0, false
	}

	return lineNumber, true
}
