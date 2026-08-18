package alpha

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/partir/core/pkg/plugin"
)

// Importer handles importing existing architecture from external data sources.
// The user never needs to understand SQL, JSON Schema, or database internals.
// Alpha connects, introspects, and generates — the user just confirms.
type Importer struct {
	workspace *Workspace
	executor  plugin.Executor
}

// NewImporter creates a new Importer.
func NewImporter(ws *Workspace, executor plugin.Executor) *Importer {
	return &Importer{workspace: ws, executor: executor}
}

// ImportFromDB connects to a Postgres/MySQL database and introspects the schema.
// It queries information_schema to discover tables, columns, types, and foreign keys,
// then uses the LLM to refine names and add descriptions.
func (i *Importer) ImportFromDB(ctx context.Context, project, dsn string) (*Rulebook, error) {
	log.Printf("[alpha-import] Connecting to database for project %s", project)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Introspect tables
	tables, err := i.introspectTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("introspect tables: %w", err)
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables found in database")
	}

	log.Printf("[alpha-import] Found %d tables", len(tables))

	// Introspect columns for each table
	schemas := make(map[string]interface{})
	for _, table := range tables {
		columns, err := i.introspectColumns(ctx, db, table)
		if err != nil {
			log.Printf("[alpha-import] Warning: could not introspect %s: %v", table, err)
			continue
		}

		schema := i.columnsToJSONSchema(table, columns)
		schemas[table+".json"] = schema
	}

	// Introspect foreign keys for relationships
	fks, err := i.introspectForeignKeys(ctx, db)
	if err != nil {
		log.Printf("[alpha-import] Warning: could not introspect foreign keys: %v", err)
	}

	// Initialize workspace and write schemas
	if !i.workspace.Exists(project) {
		if _, err := i.workspace.Init(project); err != nil {
			return nil, fmt.Errorf("init workspace: %w", err)
		}
	}

	// Write each schema to the workspace
	for name, schema := range schemas {
		schemaPath := filepath.Join(i.workspace.BaseDir, project, "schemas", name)
		data, _ := json.MarshalIndent(schema, "", "  ")
		if err := os.WriteFile(schemaPath, data, 0644); err != nil {
			log.Printf("[alpha-import] Warning: could not write schema %s: %v", name, err)
		}
	}

	// Write foreign keys as connections
	if len(fks) > 0 {
		connectionsPath := filepath.Join(i.workspace.BaseDir, project, "connections.json")
		data, _ := json.MarshalIndent(map[string]interface{}{
			"connections": fks,
		}, "", "  ")
		os.WriteFile(connectionsPath, data, 0644)
	}

	// Load and refine via LLM
	rb, err := i.workspace.Load(project)
	if err != nil {
		return nil, fmt.Errorf("load workspace after import: %w", err)
	}

	// Use LLM to refine the generated schemas
	if i.executor != nil {
		i.refineWithLLM(ctx, project, rb)
	}

	log.Printf("[alpha-import] Import complete: %d schemas, %d foreign keys", len(schemas), len(fks))
	return rb, nil
}

// ImportFromOpenAPI parses an OpenAPI/Swagger specification and generates schemas.
func (i *Importer) ImportFromOpenAPI(ctx context.Context, project, specPath string) (*Rulebook, error) {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI spec: %w", err)
	}

	// Initialize workspace
	if !i.workspace.Exists(project) {
		if _, err := i.workspace.Init(project); err != nil {
			return nil, fmt.Errorf("init workspace: %w", err)
		}
	}

	// Use LLM to extract schemas from OpenAPI spec
	if i.executor == nil {
		return nil, fmt.Errorf("executor required for OpenAPI import")
	}

	prompt := fmt.Sprintf(`Extract all data model schemas from this OpenAPI specification.
For each schema/model in the spec, output a JSON Schema in a tagged code block.

Use this format for each model:
`+"```alpha:schema:model_name"+`
{ JSON Schema here }
`+"```"+`

OpenAPI Spec:
%s`, string(content))

	resp, err := i.executor.Execute(ctx, &plugin.ExecuteRequest{
		Prompt:       prompt,
		SystemPrompt: "You are a schema extraction expert. Convert OpenAPI schemas to JSON Schema format.",
		MaxTokens:    8192,
		Temperature:  0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM extraction: %w", err)
	}

	// The extractor in the interview package will handle parsing
	// For now, write the raw output so the user can review
	outputPath := filepath.Join(i.workspace.BaseDir, project, "openapi_import.md")
	os.WriteFile(outputPath, []byte(resp.Output), 0644)

	return i.workspace.Load(project)
}

// ImportCatalogFromCSV reads a CSV file and creates a catalog from it.
func (i *Importer) ImportCatalogFromCSV(ctx context.Context, project, csvPath, catalogName string) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read CSV header: %w", err)
	}

	// Read all rows
	var catalogs []Catalog
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read CSV row: %w", err)
		}

		props := make(map[string]interface{})
		for j, header := range headers {
			if j < len(row) {
				props[header] = row[j]
			}
		}

		catalog := Catalog{
			ID:         fmt.Sprintf("%s_%d", catalogName, len(catalogs)),
			Type:       catalogName,
			Name:       row[0], // use first column as name
			Properties: props,
		}
		catalogs = append(catalogs, catalog)
	}

	// Write to workspace
	if !i.workspace.Exists(project) {
		if _, err := i.workspace.Init(project); err != nil {
			return fmt.Errorf("init workspace: %w", err)
		}
	}

	catalogPath := filepath.Join(i.workspace.BaseDir, project, "catalogs", catalogName+".json")
	data, _ := json.MarshalIndent(catalogs, "", "  ")
	if err := os.WriteFile(catalogPath, data, 0644); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}

	log.Printf("[alpha-import] CSV catalog imported: %s (%d items)", catalogName, len(catalogs))
	return nil
}

// ImportCatalogFromURL registers an external catalog source (S3, Postgres, API).
func (i *Importer) ImportCatalogFromURL(ctx context.Context, project, url, catalogName string) error {
	if !i.workspace.Exists(project) {
		if _, err := i.workspace.Init(project); err != nil {
			return fmt.Errorf("init workspace: %w", err)
		}
	}

	source := CatalogSource{
		ID:   catalogName,
		Name: catalogName,
		Type: detectSourceType(url),
		URI:  url,
	}

	// Write catalog source config
	sourcePath := filepath.Join(i.workspace.BaseDir, project, "catalogs", catalogName+"_source.json")
	data, _ := json.MarshalIndent(source, "", "  ")
	if err := os.WriteFile(sourcePath, data, 0644); err != nil {
		return fmt.Errorf("write catalog source: %w", err)
	}

	log.Printf("[alpha-import] External catalog linked: %s → %s (%s)", catalogName, url, source.Type)
	return nil
}

// ---- Internal helpers ----

// columnInfo holds introspected column metadata.
type columnInfo struct {
	Name       string
	DataType   string
	IsNullable bool
	Default    *string
}

// fkInfo holds introspected foreign key metadata.
type fkInfo struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
}

func (i *Importer) introspectTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (i *Importer) introspectColumns(ctx context.Context, db *sql.DB, table string) ([]columnInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT column_name, data_type, is_nullable, column_default
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = $1
		 ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []columnInfo
	for rows.Next() {
		var col columnInfo
		var nullable string
		var def sql.NullString
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &def); err != nil {
			return nil, err
		}
		col.IsNullable = nullable == "YES"
		if def.Valid {
			col.Default = &def.String
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func (i *Importer) introspectForeignKeys(ctx context.Context, db *sql.DB) ([]fkInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT
			kcu.table_name AS from_table,
			kcu.column_name AS from_column,
			ccu.table_name AS to_table,
			ccu.column_name AS to_column
		 FROM information_schema.table_constraints tc
		 JOIN information_schema.key_column_usage kcu
		   ON tc.constraint_name = kcu.constraint_name
		 JOIN information_schema.constraint_column_usage ccu
		   ON tc.constraint_name = ccu.constraint_name
		 WHERE tc.constraint_type = 'FOREIGN KEY'
		   AND tc.table_schema = 'public'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []fkInfo
	for rows.Next() {
		var fk fkInfo
		if err := rows.Scan(&fk.FromTable, &fk.FromColumn, &fk.ToTable, &fk.ToColumn); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

// columnsToJSONSchema converts introspected columns to a JSON Schema object.
func (i *Importer) columnsToJSONSchema(table string, columns []columnInfo) map[string]interface{} {
	properties := make(map[string]interface{})
	var required []string

	for _, col := range columns {
		prop := map[string]interface{}{
			"type": sqlTypeToJSONType(col.DataType),
		}
		properties[col.Name] = prop

		if !col.IsNullable {
			required = append(required, col.Name)
		}
	}

	return map[string]interface{}{
		"title":      table,
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// sqlTypeToJSONType maps SQL data types to JSON Schema types.
func sqlTypeToJSONType(sqlType string) string {
	switch strings.ToLower(sqlType) {
	case "integer", "smallint", "bigint", "serial", "bigserial":
		return "integer"
	case "numeric", "decimal", "real", "double precision", "float":
		return "number"
	case "boolean":
		return "boolean"
	case "json", "jsonb":
		return "object"
	case "array":
		return "array"
	default:
		return "string"
	}
}

// refineWithLLM sends the generated schemas to the LLM for name cleanup and descriptions.
func (i *Importer) refineWithLLM(ctx context.Context, project string, rb *Rulebook) {
	// This is a best-effort refinement — don't fail if LLM is unavailable
	schemasJSON, err := json.MarshalIndent(rb.Schemas, "", "  ")
	if err != nil {
		return
	}

	resp, err := i.executor.Execute(ctx, &plugin.ExecuteRequest{
		Prompt: fmt.Sprintf(`I imported these schemas from an existing database.
Please review and suggest improvements:
- Clean up column names (e.g., "usr_nm" → "user_name")
- Add descriptions for each field
- Flag any potential issues

Schemas:
%s`, string(schemasJSON)),
		SystemPrompt: "You are a data model reviewer. Suggest improvements to imported database schemas.",
		MaxTokens:    2048,
		Temperature:  0.3,
	})
	if err != nil {
		log.Printf("[alpha-import] LLM refinement skipped: %v", err)
		return
	}

	// Save refinement suggestions
	suggestPath := filepath.Join(i.workspace.BaseDir, project, "import_suggestions.md")
	os.WriteFile(suggestPath, []byte(resp.Output), 0644)
	log.Printf("[alpha-import] LLM refinement suggestions saved to import_suggestions.md")
}

// detectSourceType guesses the source type from a URL/connection string.
func detectSourceType(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://"):
		return "postgres"
	case strings.HasPrefix(lower, "s3://") || strings.Contains(lower, ".s3."):
		return "s3"
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		if strings.Contains(lower, ".csv") {
			return "csv_url"
		}
		return "api"
	default:
		return "csv_url"
	}
}
