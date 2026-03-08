package scanner

import (
	"strings"
	"testing"
)

func TestCountTokens(t *testing.T) {
	text := "Hello, world! This is a test."
	count, err := CountTokens(text)
	if err != nil {
		t.Fatalf("CountTokens failed: %v", err)
	}
	if count == 0 {
		t.Error("expected non-zero token count")
	}
	// "Hello, world! This is a test." should be roughly 8-10 tokens
	if count < 5 || count > 15 {
		t.Errorf("unexpected token count %d for simple sentence", count)
	}
}

func TestCountTokensEmpty(t *testing.T) {
	count, err := CountTokens("")
	if err != nil {
		t.Fatalf("CountTokens failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", count)
	}
}

func TestChunkFileSmall(t *testing.T) {
	pf := &ParsedFile{
		Path:      "small.go",
		Language:  "Go",
		LineCount: 50,
	}
	content := []byte(strings.Repeat("x := 1\n", 50))

	chunks, err := ChunkFile(pf, content)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small file, got %d", len(chunks))
	}
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 50 {
		t.Errorf("expected chunk lines 1-50, got %d-%d", chunks[0].StartLine, chunks[0].EndLine)
	}
}

func TestChunkFileLargeNoSymbols(t *testing.T) {
	pf := &ParsedFile{
		Path:      "large.txt",
		Language:  "Other",
		LineCount: 300,
	}
	content := []byte(strings.Repeat("line\n", 300))

	chunks, err := ChunkFile(pf, content)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	// No symbols to split at, so should be a single chunk
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (no symbols to split at), got %d", len(chunks))
	}
}

func TestChunkFileLargeWithSymbols(t *testing.T) {
	pf := &ParsedFile{
		Path:      "large.go",
		Language:  "Go",
		LineCount: 400,
		Functions: []Symbol{
			{Name: "Func1", StartLine: 1, EndLine: 100},
			{Name: "Func2", StartLine: 101, EndLine: 200},
			{Name: "Func3", StartLine: 201, EndLine: 300},
			{Name: "Func4", StartLine: 301, EndLine: 400},
		},
	}

	var lines []string
	for i := 0; i < 400; i++ {
		lines = append(lines, "x := 1")
	}
	content := []byte(strings.Join(lines, "\n"))

	chunks, err := ChunkFile(pf, content)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks for 400-line file with symbols, got %d", len(chunks))
	}

	// Verify all chunks have tokens counted
	for i, c := range chunks {
		if c.Tokens == 0 {
			t.Errorf("chunk %d has 0 tokens", i)
		}
	}
}

func TestCalculateBudget(t *testing.T) {
	chunks := []Chunk{
		{Tokens: 1000},
		{Tokens: 2000},
		{Tokens: 3000},
	}
	docChunks := []Chunk{
		{Tokens: 500},
	}

	budget := CalculateBudget(chunks, docChunks)

	if budget.Summarization != 6000 {
		t.Errorf("expected summarization 6000, got %d", budget.Summarization)
	}
	if budget.DocIngestion != 500 {
		t.Errorf("expected doc ingestion 500, got %d", budget.DocIngestion)
	}
	if budget.TotalInput == 0 {
		t.Error("expected non-zero total input")
	}
	if budget.EstimatedOutput == 0 {
		t.Error("expected non-zero estimated output")
	}

	// Output should be ~12.5% of input
	ratio := float64(budget.EstimatedOutput) / float64(budget.TotalInput)
	if ratio < 0.10 || ratio > 0.15 {
		t.Errorf("output/input ratio %.2f outside expected range 0.10-0.15", ratio)
	}
}
