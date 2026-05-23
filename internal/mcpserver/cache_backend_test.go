// SPDX-License-Identifier: Apache-2.0

package mcpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rokasklive/opengrok-go-mcp/internal/cache"
	"github.com/rokasklive/opengrok-go-mcp/internal/opengrok"
)

func TestCachingBackendListProjectsHitAndMiss(t *testing.T) {
	fake := &fakeBackend{projects: []string{"platform", "core"}}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	projects, err := backend.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects error = %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	projects2, err := backend.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects error = %v", err)
	}
	if len(projects2) != 2 {
		t.Fatalf("expected 2 projects on cache hit, got %d", len(projects2))
	}
}

func TestCachingBackendListFilesHitAndMiss(t *testing.T) {
	fake := &fakeBackend{
		fileEntries: []opengrok.FileEntry{
			{Path: "src/main.go", IsDirectory: false},
		},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	entries, err := backend.ListFiles(ctx, "platform", "src")
	if err != nil {
		t.Fatalf("ListFiles error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entries2, err := backend.ListFiles(ctx, "platform", "src")
	if err != nil {
		t.Fatalf("ListFiles error = %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1 entry on cache hit, got %d", len(entries2))
	}
}

func TestCachingBackendPreservesListFilesTruncationMetadata(t *testing.T) {
	fake := &fakeBackend{
		fileEntries:       []opengrok.FileEntry{{Path: "src/main.go"}},
		fileListTruncated: true,
	}
	backend := NewCachingBackend(fake, cache.New(time.Minute), 100)
	ctx := context.Background()

	_, truncated, err := backend.ListFilesWithMetadata(ctx, "platform", "")
	if err != nil {
		t.Fatalf("ListFilesWithMetadata first error = %v", err)
	}
	if !truncated {
		t.Fatal("first truncated = false, want true")
	}

	fake.fileListTruncated = false
	_, truncated, err = backend.ListFilesWithMetadata(ctx, "platform", "")
	if err != nil {
		t.Fatalf("ListFilesWithMetadata cached error = %v", err)
	}
	if !truncated {
		t.Fatal("cached truncated = false, want cached true")
	}
}

func TestCachingBackendSearchNotCached(t *testing.T) {
	fake := &fakeBackend{
		searchResult: opengrok.SearchResult{TotalHits: 5},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	req := opengrok.SearchRequest{Projects: []string{"platform"}, Query: "test"}
	result, err := backend.Search(ctx, req)
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if result.TotalHits != 5 {
		t.Fatalf("expected 5 hits, got %d", result.TotalHits)
	}

	if c.Size() != 0 {
		t.Fatalf("expected cache size 0 (search not cached), got %d", c.Size())
	}
}

func TestCachingBackendFileContentHitAndMiss(t *testing.T) {
	fake := &fakeBackend{fileContent: "package main"}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	content, err := backend.FileContent(ctx, "platform", "src/main.go")
	if err != nil {
		t.Fatalf("FileContent error = %v", err)
	}
	if content != "package main" {
		t.Fatalf("expected 'package main', got %q", content)
	}

	content2, err := backend.FileContent(ctx, "platform", "src/main.go")
	if err != nil {
		t.Fatalf("FileContent error = %v", err)
	}
	if content2 != "package main" {
		t.Fatalf("expected 'package main' on cache hit, got %q", content2)
	}
}

func TestCachingBackendGetProjectOverviewHitAndMiss(t *testing.T) {
	fake := &fakeBackend{
		projectOverview: opengrok.ProjectOverview{
			Project:    "platform",
			TotalFiles: 42,
		},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	overview, err := backend.GetProjectOverview(ctx, "platform")
	if err != nil {
		t.Fatalf("GetProjectOverview error = %v", err)
	}
	if overview.TotalFiles != 42 {
		t.Fatalf("expected 42 files, got %d", overview.TotalFiles)
	}

	overview2, err := backend.GetProjectOverview(ctx, "platform")
	if err != nil {
		t.Fatalf("GetProjectOverview error = %v", err)
	}
	if overview2.TotalFiles != 42 {
		t.Fatalf("expected 42 files on cache hit, got %d", overview2.TotalFiles)
	}
}

func TestCachingBackendMaxSizeEviction(t *testing.T) {
	fake := &fakeBackend{
		fileEntries: []opengrok.FileEntry{
			{Path: "src/main.go", IsDirectory: false},
		},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 2)
	ctx := context.Background()

	_, _ = backend.ListProjects(ctx)
	_, _ = backend.ListFiles(ctx, "p1", "")
	if c.Size() != 2 {
		t.Fatalf("expected cache size 2, got %d", c.Size())
	}

	_, _ = backend.ListFiles(ctx, "p2", "")
	if c.Size() != 2 {
		t.Fatalf("expected cache size 2 after LRU eviction, got %d", c.Size())
	}
	if _, ok := c.Get("projects:list"); ok {
		t.Fatal("least recently used cache entry survived eviction")
	}
	if _, ok := c.Get("files:p2\x00"); !ok {
		t.Fatal("fresh cache entry was evicted")
	}
}

func TestCachingBackendErrorNotCached(t *testing.T) {
	fake := &fakeBackend{projectsErr: errors.New("network error")}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	_, err := backend.ListProjects(ctx)
	if err == nil {
		t.Fatal("expected error from backend")
	}

	if c.Size() != 0 {
		t.Fatalf("expected cache size 0 (errors not cached), got %d", c.Size())
	}
}

func TestCachingBackendDifferentKeys(t *testing.T) {
	fake := &fakeBackend{
		fileContents: map[string]string{
			"platform:src/a.go": "a",
			"platform:src/b.go": "b",
		},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	a, err := backend.FileContent(ctx, "platform", "src/a.go")
	if err != nil {
		t.Fatalf("FileContent error = %v", err)
	}
	if a != "a" {
		t.Fatalf("expected 'a', got %q", a)
	}

	b, err := backend.FileContent(ctx, "platform", "src/b.go")
	if err != nil {
		t.Fatalf("FileContent error = %v", err)
	}
	if b != "b" {
		t.Fatalf("expected 'b', got %q", b)
	}

	if c.Size() != 2 {
		t.Fatalf("expected cache size 2, got %d", c.Size())
	}
}

func TestCachingBackendFileContentKeyDoesNotCollideAcrossComponentBoundary(t *testing.T) {
	fake := &fakeBackend{
		fileContents: map[string]string{
			"a:b:foo.go": "first",
		},
	}
	c := cache.New(1 * time.Minute)
	backend := NewCachingBackend(fake, c, 100)
	ctx := context.Background()

	first, err := backend.FileContent(ctx, "a", "b:foo.go")
	if err != nil {
		t.Fatalf("FileContent first error = %v", err)
	}
	if first != "first" {
		t.Fatalf("first content = %q, want first", first)
	}

	fake.fileContents["a:b:foo.go"] = "second"
	second, err := backend.FileContent(ctx, "a:b", "foo.go")
	if err != nil {
		t.Fatalf("FileContent second error = %v", err)
	}
	if second != "second" {
		t.Fatalf("second content = %q, want second", second)
	}
}

func TestCachingBackendDoesNotExposeCachedSliceMutation(t *testing.T) {
	size := int64(10)
	fake := &fakeBackend{
		projects: []string{"platform"},
		fileEntries: []opengrok.FileEntry{
			{Path: "src/main.go", Size: &size},
		},
		projectOverview: opengrok.ProjectOverview{
			TopFiles: []opengrok.FileEntry{{Path: "README.md", Size: &size}},
		},
	}
	backend := NewCachingBackend(fake, cache.New(time.Minute), 100)
	ctx := context.Background()

	_, _ = backend.ListProjects(ctx)
	projects, _ := backend.ListProjects(ctx)
	projects[0] = "mutated"
	projectsAgain, _ := backend.ListProjects(ctx)
	if projectsAgain[0] != "platform" {
		t.Fatalf("cached projects were mutated: %#v", projectsAgain)
	}

	_, _ = backend.ListFiles(ctx, "platform", "")
	entries, _ := backend.ListFiles(ctx, "platform", "")
	entries[0].Path = "mutated"
	*entries[0].Size = 99
	entriesAgain, _ := backend.ListFiles(ctx, "platform", "")
	if entriesAgain[0].Path != "src/main.go" || *entriesAgain[0].Size != 10 {
		t.Fatalf("cached file entries were mutated: %#v", entriesAgain)
	}

	_, _ = backend.GetProjectOverview(ctx, "platform")
	overview, _ := backend.GetProjectOverview(ctx, "platform")
	overview.TopFiles[0].Path = "mutated"
	*overview.TopFiles[0].Size = 99
	overviewAgain, _ := backend.GetProjectOverview(ctx, "platform")
	if overviewAgain.TopFiles[0].Path != "README.md" || *overviewAgain.TopFiles[0].Size != 10 {
		t.Fatalf("cached project overview was mutated: %#v", overviewAgain)
	}
}
