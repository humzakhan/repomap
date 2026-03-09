package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// FileEntry represents a single file in the repository.
type FileEntry struct {
	Path         string
	RelPath      string
	Language     string
	SizeBytes    int64
	IsEntryPoint bool
	SkipAnalysis bool   // true if file should not be sent to LLM
	SkipReason   string // "test", "generated", "barrel", or ""
}

// WalkResult contains the results of walking a repository.
type WalkResult struct {
	Files         []FileEntry
	LanguageStats map[string]int
	TotalFiles    int
	SkippedFiles  int
}

// Known language extensions.
var languageExtensions = map[string]string{
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".js":    "JavaScript",
	".jsx":   "JavaScript",
	".py":    "Python",
	".go":    "Go",
	".rs":    "Rust",
	".rb":    "Ruby",
	".java":  "Java",
	".json":  "JSON",
	".yaml":  "YAML",
	".yml":   "YAML",
	".toml":  "TOML",
	".md":    "Markdown",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "CSS",
	".sql":   "SQL",
	".sh":    "Shell",
	".bash":  "Shell",
	".zsh":   "Shell",
	".proto": "Protocol Buffers",
}

// Directories that are always skipped.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".next":        true,
	".nuxt":        true,
	"dist":         true,
	"build":        true,
	".terraform":   true,
}

// Entry point filename patterns (case-insensitive base name, without extension).
var entryPointNames = map[string]bool{
	"main":   true,
	"index":  true,
	"app":    true,
	"server": true,
	"cli":    true,
	"cmd":    true,
}

// Walk traverses a repository directory, respecting .gitignore rules.
func Walk(ctx context.Context, repoRoot string) (*WalkResult, error) {
	repoRoot = filepath.Clean(repoRoot)

	info, err := os.Stat(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("accessing repo root %s: %w", repoRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", repoRoot)
	}

	// Load .gitignore patterns
	ignorer, err := loadGitignore(repoRoot)
	if err != nil {
		// If .gitignore can't be loaded, proceed without it
		ignorer = nil
	}

	result := &WalkResult{
		LanguageStats: make(map[string]int),
	}

	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}

		// Skip root itself
		if relPath == "." {
			return nil
		}

		// Skip always-ignored directories
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Validate path is within repo root (no traversal)
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		absRoot, err := filepath.Abs(repoRoot)
		if err != nil {
			return nil
		}
		if !strings.HasPrefix(absPath, absRoot) {
			return nil
		}

		// Check .gitignore
		if ignorer != nil && ignorer.MatchesPath(relPath) {
			result.SkippedFiles++
			return nil
		}

		// Skip binary and very large files
		if info.Size() > 10*1024*1024 { // 10MB
			result.SkippedFiles++
			return nil
		}

		// Detect language
		ext := strings.ToLower(filepath.Ext(path))
		lang := languageExtensions[ext]
		if lang == "" {
			lang = "Other"
		}

		// Detect entry points
		baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		isEntry := entryPointNames[strings.ToLower(baseName)]

		entry := FileEntry{
			Path:         absPath,
			RelPath:      relPath,
			Language:     lang,
			SizeBytes:    info.Size(),
			IsEntryPoint: isEntry,
		}

		result.Files = append(result.Files, entry)
		result.LanguageStats[lang]++
		result.TotalFiles++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking repository %s: %w", repoRoot, err)
	}

	return result, nil
}

// loadGitignore reads .gitignore from the repo root and returns a compiled ignorer.
func loadGitignore(repoRoot string) (*gitignore.GitIgnore, error) {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return nil, nil
	}

	ig, err := gitignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil, fmt.Errorf("parsing .gitignore: %w", err)
	}

	return ig, nil
}
