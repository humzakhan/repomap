package scanner

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// Stage represents a pipeline stage for chunking.
type Stage int

const (
	// StageSummarization is the module summary stage.
	StageSummarization Stage = iota
	// StageSynthesis is the architecture synthesis stage.
	StageSynthesis
	// StageDocIngestion is the documentation ingestion stage.
	StageDocIngestion
)

// Chunk represents a portion of a file to be sent to the LLM.
type Chunk struct {
	FilePath  string `json:"file_path"`
	Stage     Stage  `json:"stage"`
	Content   string `json:"content"`
	Tokens    int    `json:"tokens"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// TokenBudget contains the estimated token usage for each pipeline stage.
type TokenBudget struct {
	Summarization   int `json:"summarization"`
	Synthesis       int `json:"synthesis"`
	DocIngestion    int `json:"doc_ingestion"`
	TotalInput      int `json:"total_input"`
	EstimatedOutput int `json:"estimated_output"`
}

// outputRatio is the estimated output-to-input token ratio.
const outputRatio = 0.125 // 12.5%

// maxChunkLines is the line threshold for splitting files into chunks.
const maxChunkLines = 200

// tokenEncoder is a lazily-initialized tiktoken encoder.
var tokenEncoder *tiktoken.Tiktoken

func getEncoder() (*tiktoken.Tiktoken, error) {
	if tokenEncoder != nil {
		return tokenEncoder, nil
	}
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("initializing tokenizer: %w", err)
	}
	tokenEncoder = enc
	return enc, nil
}

// CountTokens returns the approximate token count for a string.
func CountTokens(text string) (int, error) {
	enc, err := getEncoder()
	if err != nil {
		return 0, err
	}
	tokens := enc.Encode(text, nil, nil)
	return len(tokens), nil
}

// ChunkFile splits a parsed file into chunks suitable for LLM processing.
// Files under maxChunkLines lines are a single chunk.
// Larger files are split at function/class boundaries.
func ChunkFile(pf *ParsedFile, content []byte) ([]Chunk, error) {
	if pf.LineCount <= maxChunkLines {
		tokens, err := CountTokens(string(content))
		if err != nil {
			return nil, fmt.Errorf("counting tokens for %s: %w", pf.Path, err)
		}
		return []Chunk{{
			FilePath:  pf.Path,
			Stage:     StageSummarization,
			Content:   string(content),
			Tokens:    tokens,
			StartLine: 1,
			EndLine:   pf.LineCount,
		}}, nil
	}

	// Collect all symbol boundaries for split points
	type boundary struct {
		startLine int
		endLine   int
	}
	var boundaries []boundary
	allSymbols := append(append(pf.Functions, pf.Classes...), pf.Interfaces...)
	for _, sym := range allSymbols {
		boundaries = append(boundaries, boundary{startLine: sym.StartLine, endLine: sym.EndLine})
	}

	// If no symbols found, fall back to single chunk
	if len(boundaries) == 0 {
		tokens, err := CountTokens(string(content))
		if err != nil {
			return nil, fmt.Errorf("counting tokens for %s: %w", pf.Path, err)
		}
		return []Chunk{{
			FilePath:  pf.Path,
			Stage:     StageSummarization,
			Content:   string(content),
			Tokens:    tokens,
			StartLine: 1,
			EndLine:   pf.LineCount,
		}}, nil
	}

	// Split content into lines
	lines := splitLines(content)

	// Group symbols into chunks that don't exceed maxChunkLines
	var chunks []Chunk
	chunkStart := 1
	chunkEnd := 0

	for _, b := range boundaries {
		// If adding this symbol would exceed the limit, flush the current chunk
		if chunkEnd > 0 && b.endLine-chunkStart+1 > maxChunkLines {
			chunk, err := makeChunk(pf.Path, lines, chunkStart, chunkEnd)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, chunk)
			chunkStart = chunkEnd + 1
		}
		chunkEnd = b.endLine
	}

	// Flush remaining content
	if chunkStart <= pf.LineCount {
		finalEnd := pf.LineCount
		if chunkEnd > chunkStart {
			finalEnd = max(chunkEnd, pf.LineCount)
		}
		chunk, err := makeChunk(pf.Path, lines, chunkStart, finalEnd)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// CalculateBudget computes the token budget from a set of chunks.
func CalculateBudget(chunks []Chunk, docChunks []Chunk) TokenBudget {
	budget := TokenBudget{}

	for _, c := range chunks {
		budget.Summarization += c.Tokens
	}

	for _, c := range docChunks {
		budget.DocIngestion += c.Tokens
	}

	// Synthesis input is estimated as ~20% of summarization output
	budget.Synthesis = int(float64(budget.Summarization) * outputRatio * 1.5)

	budget.TotalInput = budget.Summarization + budget.Synthesis + budget.DocIngestion
	budget.EstimatedOutput = int(float64(budget.TotalInput) * outputRatio)

	return budget
}

func makeChunk(filePath string, lines []string, startLine int, endLine int) (Chunk, error) {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var content string
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		content += lines[i] + "\n"
	}

	tokens, err := CountTokens(content)
	if err != nil {
		return Chunk{}, fmt.Errorf("counting tokens for %s lines %d-%d: %w", filePath, startLine, endLine, err)
	}

	return Chunk{
		FilePath:  filePath,
		Stage:     StageSummarization,
		Content:   content,
		Tokens:    tokens,
		StartLine: startLine,
		EndLine:   endLine,
	}, nil
}

func splitLines(content []byte) []string {
	var lines []string
	start := 0
	for i, b := range content {
		if b == '\n' {
			lines = append(lines, string(content[start:i]))
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, string(content[start:]))
	}
	return lines
}
