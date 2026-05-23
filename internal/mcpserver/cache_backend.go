package mcpserver

import (
	"context"

	"github.com/rokasklive/opengrok-go-mcp/internal/cache"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

var _ Backend = (*CachingBackend)(nil)

// CachingBackend wraps a Backend with an in-memory TTL cache.
type CachingBackend struct {
	backend Backend
	cache   *cache.Cache
	maxSize int
}

// NewCachingBackend creates a caching wrapper around the given backend.
func NewCachingBackend(backend Backend, cache *cache.Cache, maxSize int) *CachingBackend {
	return &CachingBackend{backend: backend, cache: cache, maxSize: maxSize}
}

func (c *CachingBackend) checkEviction() {
	c.cache.TrimToSize(c.maxSize)
}

// ListProjects returns the list of projects, caching the result.
func (c *CachingBackend) ListProjects(ctx context.Context) ([]string, error) {
	const key = "projects:list"
	if val, ok := c.cache.Get(key); ok {
		if projects, ok := val.([]string); ok {
			return cloneStrings(projects), nil
		}
	}

	projects, err := c.backend.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	c.cache.Set(key, cloneStrings(projects))
	c.checkEviction()
	return cloneStrings(projects), nil
}

// ListFiles returns files for a project path, caching the result.
func (c *CachingBackend) ListFiles(ctx context.Context, project string, path string) ([]opengrok.FileEntry, error) {
	entries, _, err := c.ListFilesWithMetadata(ctx, project, path)
	return entries, err
}

type cachedFileList struct {
	entries   []opengrok.FileEntry
	truncated bool
}

// ListFilesWithMetadata preserves the truncation signal for callers that
// understand it while still satisfying the legacy Backend method.
func (c *CachingBackend) ListFilesWithMetadata(
	ctx context.Context,
	project string,
	path string,
) ([]opengrok.FileEntry, bool, error) {
	key := "files:" + project + "\x00" + path
	if val, ok := c.cache.Get(key); ok {
		if result, ok := val.(cachedFileList); ok {
			return cloneFileEntries(result.entries), result.truncated, nil
		}
	}

	entries, truncated, err := listFilesWithMetadata(ctx, c.backend, project, path)
	if err != nil {
		return nil, false, err
	}

	c.cache.Set(key, cachedFileList{
		entries:   cloneFileEntries(entries),
		truncated: truncated,
	})
	c.checkEviction()
	return cloneFileEntries(entries), truncated, nil
}

// Search is not cached because results are too variable.
func (c *CachingBackend) Search(ctx context.Context, req opengrok.SearchRequest) (opengrok.SearchResult, error) {
	return c.backend.Search(ctx, req)
}

// FileContent returns file content, caching the result.
func (c *CachingBackend) FileContent(ctx context.Context, project string, filePath string) (string, error) {
	key := "content:" + project + "\x00" + filePath
	if val, ok := c.cache.Get(key); ok {
		if content, ok := val.(string); ok {
			return content, nil
		}
	}

	content, err := c.backend.FileContent(ctx, project, filePath)
	if err != nil {
		return "", err
	}

	c.cache.Set(key, content)
	c.checkEviction()
	return content, nil
}

// GetProjectOverview returns project overview, caching the result.
func (c *CachingBackend) GetProjectOverview(ctx context.Context, project string) (opengrok.ProjectOverview, error) {
	key := "overview:" + project
	if val, ok := c.cache.Get(key); ok {
		if overview, ok := val.(opengrok.ProjectOverview); ok {
			return cloneProjectOverview(overview), nil
		}
	}

	overview, err := c.backend.GetProjectOverview(ctx, project)
	if err != nil {
		return opengrok.ProjectOverview{}, err
	}

	c.cache.Set(key, cloneProjectOverview(overview))
	c.checkEviction()
	return cloneProjectOverview(overview), nil
}

func cloneStrings(values []string) []string {
	return append([]string{}, values...)
}

func cloneFileEntries(entries []opengrok.FileEntry) []opengrok.FileEntry {
	cloned := make([]opengrok.FileEntry, len(entries))
	for i, entry := range entries {
		cloned[i] = entry
		if entry.Size != nil {
			size := *entry.Size
			cloned[i].Size = &size
		}
	}
	return cloned
}

func cloneProjectOverview(overview opengrok.ProjectOverview) opengrok.ProjectOverview {
	overview.TopDirs = cloneFileEntries(overview.TopDirs)
	overview.TopFiles = cloneFileEntries(overview.TopFiles)
	return overview
}
