package scanner

import (
	"path/filepath"
	"strings"
)

// Test file name suffixes across supported languages.
var testSuffixes = []string{
	"_test.go",
	".test.ts", ".test.tsx", ".test.js", ".test.jsx",
	".spec.ts", ".spec.tsx", ".spec.js", ".spec.jsx",
	"_test.py", "_test.rs",
}

// Directories that exclusively contain test files.
var testDirs = map[string]bool{
	"__tests__": true,
	"test":      true,
	"tests":     true,
	"spec":      true,
}

// Markers found in the first bytes of generated files.
var generatedMarkers = []string{
	"Code generated",
	"DO NOT EDIT",
	"auto-generated",
	"AUTO-GENERATED",
	"@generated",
	"This file is generated",
	"THIS FILE IS GENERATED",
}

// ClassifyFile determines whether a file should be skipped for LLM analysis.
// It sets SkipAnalysis and SkipReason on the entry in-place.
// The file still appears in the dependency graph — only LLM chunking is skipped.
func ClassifyFile(entry *FileEntry, content []byte) {
	if isTestFile(entry.RelPath) {
		entry.SkipAnalysis = true
		entry.SkipReason = "test"
		return
	}
	if isGeneratedFile(content) {
		entry.SkipAnalysis = true
		entry.SkipReason = "generated"
		return
	}
	if isBarrelFile(entry.Language, content) {
		entry.SkipAnalysis = true
		entry.SkipReason = "barrel"
		return
	}
}

// isTestFile returns true if the file path indicates a test file,
// either by filename suffix or by residing in a test directory.
func isTestFile(relPath string) bool {
	base := filepath.Base(relPath)

	for _, suffix := range testSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	if strings.HasPrefix(base, "test_") {
		return true
	}

	parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	for _, part := range parts {
		if testDirs[strings.ToLower(part)] {
			return true
		}
	}
	return false
}

// isGeneratedFile checks the first 1KB of a file for common code-generation markers.
func isGeneratedFile(content []byte) bool {
	header := content
	if len(header) > 1024 {
		header = header[:1024]
	}
	headerStr := string(header)
	for _, marker := range generatedMarkers {
		if strings.Contains(headerStr, marker) {
			return true
		}
	}
	return false
}

// isReexportLine returns true if the line is a re-export statement like:
//
//	export { Foo } from './foo';
//	export * from './bar';
//	export { default as Baz } from './baz';
//
// It returns false for declaration exports like:
//
//	export class Foo {}
//	export function bar() {}
//	export const x = 1;
func isReexportLine(trimmed string) bool {
	// "export * from" — wildcard re-export
	if strings.HasPrefix(trimmed, "export * from") || strings.HasPrefix(trimmed, "export *from") {
		return true
	}
	// "export { ... } from" or "export{ ... } from" — named re-export
	if (strings.HasPrefix(trimmed, "export {") || strings.HasPrefix(trimmed, "export{")) &&
		strings.Contains(trimmed, "} from") {
		return true
	}
	// "export type { ... } from" — type re-export
	if strings.HasPrefix(trimmed, "export type {") && strings.Contains(trimmed, "} from") {
		return true
	}
	return false
}

// isBarrelFile detects TypeScript/JavaScript files that only re-export symbols
// with no other logic. These are common index.ts files with just
// `export { X } from './foo'` lines.
func isBarrelFile(language string, content []byte) bool {
	if language != "TypeScript" && language != "JavaScript" {
		return false
	}

	lines := strings.Split(string(content), "\n")
	hasExport := false
	inBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track block comments
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
			continue
		}

		// Skip empty lines and single-line comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Allow "use strict" directives
		if trimmed == `"use strict";` || trimmed == `'use strict';` {
			continue
		}

		// Allow re-export statements: export { X } from './x' and export * from './x'
		// Reject declarations: export class, export function, export const, export default, export type (with body)
		if strings.HasPrefix(trimmed, "export ") || strings.HasPrefix(trimmed, "export{") {
			if isReexportLine(trimmed) {
				hasExport = true
				continue
			}
			// It's a declaration export, not a barrel
			return false
		}
		if strings.HasPrefix(trimmed, "import ") {
			continue
		}

		// Any other statement means this is not a barrel file
		return false
	}
	return hasExport
}
