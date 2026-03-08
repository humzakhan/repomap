package scanner

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitMetadata contains repository-level git information.
type GitMetadata struct {
	Churn   map[string]int       `json:"churn"`    // filepath -> commit count
	Authors map[string][]string  `json:"authors"`   // filepath -> unique authors
	LastMod map[string]time.Time `json:"last_mod"`  // filepath -> last modified time
}

// ExtractGitMetadata retrieves git history metadata for all files in the repo.
func ExtractGitMetadata(ctx context.Context, repoRoot string) (*GitMetadata, error) {
	meta := &GitMetadata{
		Churn:   make(map[string]int),
		Authors: make(map[string][]string),
		LastMod: make(map[string]time.Time),
	}

	// Get commit counts and authors per file using git log
	if err := extractChurnAndAuthors(ctx, repoRoot, meta); err != nil {
		return nil, fmt.Errorf("extracting churn data: %w", err)
	}

	// Get last modified timestamps
	if err := extractLastModified(ctx, repoRoot, meta); err != nil {
		return nil, fmt.Errorf("extracting last modified: %w", err)
	}

	return meta, nil
}

func extractChurnAndAuthors(ctx context.Context, repoRoot string, meta *GitMetadata) error {
	// git log --format='%H %aN' --name-only produces:
	// <hash> <author>
	// <empty line>
	// <file1>
	// <file2>
	// ...
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "log", "--format=%aN", "--name-only")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running git log: %w", err)
	}

	authorSets := make(map[string]map[string]bool) // file -> set of authors

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var currentAuthor string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// If the line doesn't contain a path separator and could be an author name,
		// treat it as author. Otherwise it's a file path.
		// Heuristic: file paths typically contain . or / while author names don't contain /
		if !strings.Contains(line, "/") && !strings.Contains(line, ".") {
			currentAuthor = line
			continue
		}

		// It's a file path
		filePath := line
		meta.Churn[filePath]++

		if currentAuthor != "" {
			if authorSets[filePath] == nil {
				authorSets[filePath] = make(map[string]bool)
			}
			authorSets[filePath][currentAuthor] = true
		}
	}

	// Convert author sets to slices
	for file, authors := range authorSets {
		for author := range authors {
			meta.Authors[file] = append(meta.Authors[file], author)
		}
	}

	return nil
}

func extractLastModified(ctx context.Context, repoRoot string, meta *GitMetadata) error {
	// Get the last commit timestamp for each tracked file
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("listing tracked files: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		filePath := scanner.Text()
		if filePath == "" {
			continue
		}

		ts, err := getFileLastModified(ctx, repoRoot, filePath)
		if err != nil {
			continue // skip files where we can't get the timestamp
		}
		meta.LastMod[filePath] = ts
	}

	return nil
}

func getFileLastModified(ctx context.Context, repoRoot string, filePath string) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "log", "-1", "--format=%at", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("getting last modified for %s: %w", filePath, err)
	}

	tsStr := strings.TrimSpace(string(out))
	if tsStr == "" {
		return time.Time{}, fmt.Errorf("no commits for %s", filePath)
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing timestamp for %s: %w", filePath, err)
	}

	return time.Unix(ts, 0).UTC(), nil
}
