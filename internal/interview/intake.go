package interview

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/partir/core/pkg/plugin"
)

// IntakeResult contains the processed content from an uploaded file.
type IntakeResult struct {
	// SourcePath is the original file path
	SourcePath string `json:"source_path"`

	// FileType is the detected file type (document, diagram, spreadsheet, code)
	FileType string `json:"file_type"`

	// Summary is an LLM-generated summary of the content
	Summary string `json:"summary"`

	// ExtractedData contains any structured data found in the file
	ExtractedData map[string]interface{} `json:"extracted_data,omitempty"`

	// RawContent is the raw text content (for text-based files)
	RawContent string `json:"raw_content,omitempty"`
}

// IntakeProcessor handles file uploads during interview sessions.
// It reads documents, diagrams, spreadsheets, and code files, then
// summarizes them via the LLM for injection into the conversation context.
type IntakeProcessor struct {
	executor plugin.Executor
}

// NewIntakeProcessor creates a new file intake processor.
func NewIntakeProcessor(executor plugin.Executor) *IntakeProcessor {
	return &IntakeProcessor{executor: executor}
}

// ProcessFile detects the file type and processes it accordingly.
func (p *IntakeProcessor) ProcessFile(ctx context.Context, path string) (*IntakeResult, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".md", ".txt", ".doc", ".docx", ".pdf":
		return p.ProcessDocument(ctx, path)
	case ".csv", ".xlsx", ".xls", ".tsv":
		return p.ProcessSpreadsheet(ctx, path)
	case ".png", ".jpg", ".jpeg", ".svg", ".webp":
		return p.ProcessImage(ctx, path)
	case ".sql", ".prisma", ".graphql", ".proto":
		return p.ProcessCode(ctx, path)
	case ".json", ".yaml", ".yml":
		return p.ProcessStructured(ctx, path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ProcessDocument reads text documents and summarizes them via the LLM.
func (p *IntakeProcessor) ProcessDocument(ctx context.Context, path string) (*IntakeResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read document %s: %w", path, err)
	}

	// For now, handle text-based formats. PDF/DOCX parsing is future work.
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".pdf" || ext == ".docx" || ext == ".doc" {
		return &IntakeResult{
			SourcePath: path,
			FileType:   "document",
			Summary:    fmt.Sprintf("Binary document detected (%s). Full parsing requires additional libraries — content noted for manual review.", ext),
			RawContent: fmt.Sprintf("[Binary file: %s — %d bytes]", filepath.Base(path), len(content)),
		}, nil
	}

	// Summarize via LLM
	summary, err := p.summarizeContent(ctx, string(content), "document")
	if err != nil {
		// Fall back to raw content if LLM fails
		summary = fmt.Sprintf("Document: %s (%d bytes)", filepath.Base(path), len(content))
	}

	return &IntakeResult{
		SourcePath: path,
		FileType:   "document",
		Summary:    summary,
		RawContent: string(content),
	}, nil
}

// ProcessSpreadsheet reads CSV/TSV files and extracts column structure.
func (p *IntakeProcessor) ProcessSpreadsheet(ctx context.Context, path string) (*IntakeResult, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// Handle Excel files as future work
	if ext == ".xlsx" || ext == ".xls" {
		return &IntakeResult{
			SourcePath: path,
			FileType:   "spreadsheet",
			Summary:    "Excel file detected. Full parsing requires additional libraries — please export as CSV for best results.",
		}, nil
	}

	// Parse CSV/TSV
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open spreadsheet %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	if ext == ".tsv" {
		reader.Comma = '\t'
	}

	// Read header + first 10 rows for analysis
	var rows [][]string
	for i := 0; i < 11; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV: %w", err)
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return &IntakeResult{
			SourcePath: path,
			FileType:   "spreadsheet",
			Summary:    "Empty spreadsheet",
		}, nil
	}

	// Build preview for LLM
	headers := rows[0]
	dataRows := rows[1:]

	var sb strings.Builder
	sb.WriteString("Spreadsheet columns: " + strings.Join(headers, ", ") + "\n")
	sb.WriteString(fmt.Sprintf("Sample rows (%d shown):\n", len(dataRows)))
	for _, row := range dataRows {
		sb.WriteString("  " + strings.Join(row, " | ") + "\n")
	}

	summary, err := p.summarizeContent(ctx, sb.String(), "spreadsheet")
	if err != nil {
		summary = fmt.Sprintf("CSV with %d columns: %s", len(headers), strings.Join(headers, ", "))
	}

	return &IntakeResult{
		SourcePath: path,
		FileType:   "spreadsheet",
		Summary:    summary,
		ExtractedData: map[string]interface{}{
			"headers":    headers,
			"row_count":  len(dataRows),
			"sample_row": dataRows[0],
		},
		RawContent: sb.String(),
	}, nil
}

// ProcessImage handles image files (sketches, diagrams, screenshots).
// Full vision processing requires a multimodal LLM; for now we note the file.
func (p *IntakeProcessor) ProcessImage(ctx context.Context, path string) (*IntakeResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat image %s: %w", path, err)
	}

	return &IntakeResult{
		SourcePath: path,
		FileType:   "diagram",
		Summary: fmt.Sprintf("Image uploaded: %s (%d KB). "+
			"Visual analysis requires a multimodal model. "+
			"Please describe what this image shows so I can incorporate it.",
			filepath.Base(path), info.Size()/1024),
	}, nil
}

// ProcessCode reads code files (SQL, Prisma, GraphQL, Proto) and extracts schema info.
func (p *IntakeProcessor) ProcessCode(ctx context.Context, path string) (*IntakeResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read code file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	fileType := "code"
	switch ext {
	case ".sql":
		fileType = "sql_schema"
	case ".prisma":
		fileType = "prisma_schema"
	case ".graphql":
		fileType = "graphql_schema"
	case ".proto":
		fileType = "protobuf_schema"
	}

	summary, err := p.summarizeContent(ctx, string(content), fileType)
	if err != nil {
		summary = fmt.Sprintf("Code file: %s (%d bytes)", filepath.Base(path), len(content))
	}

	return &IntakeResult{
		SourcePath: path,
		FileType:   fileType,
		Summary:    summary,
		RawContent: string(content),
	}, nil
}

// ProcessStructured reads JSON/YAML files and parses them.
func (p *IntakeProcessor) ProcessStructured(ctx context.Context, path string) (*IntakeResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read structured file %s: %w", path, err)
	}

	// Try to parse as JSON
	var parsed interface{}
	if err := json.Unmarshal(content, &parsed); err != nil {
		// Not JSON — treat as YAML text
		return &IntakeResult{
			SourcePath: path,
			FileType:   "structured",
			Summary:    fmt.Sprintf("Config file: %s", filepath.Base(path)),
			RawContent: string(content),
		}, nil
	}

	summary, err := p.summarizeContent(ctx, string(content), "structured_data")
	if err != nil {
		summary = fmt.Sprintf("Structured data: %s", filepath.Base(path))
	}

	data, _ := parsed.(map[string]interface{})

	return &IntakeResult{
		SourcePath:    path,
		FileType:      "structured",
		Summary:       summary,
		ExtractedData: data,
		RawContent:    string(content),
	}, nil
}

// summarizeContent uses the LLM to generate a concise summary of uploaded content.
func (p *IntakeProcessor) summarizeContent(ctx context.Context, content, fileType string) (string, error) {
	// Truncate very large content
	if len(content) > 8000 {
		content = content[:8000] + "\n\n[... truncated ...]"
	}

	prompt := fmt.Sprintf(`Summarize this %s for the purpose of understanding a software project's architecture.
Focus on: data models, business rules, technology choices, constraints, and integration points.
Be concise — 2-3 sentences max.

Content:
%s`, fileType, content)

	resp, err := p.executor.Execute(ctx, &plugin.ExecuteRequest{
		Prompt:       prompt,
		SystemPrompt: "You are a concise technical summarizer. Extract key architectural information from uploaded content.",
		MaxTokens:    256,
		Temperature:  0.3,
	})
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf("LLM error: %s", resp.Error)
	}

	return resp.Output, nil
}
