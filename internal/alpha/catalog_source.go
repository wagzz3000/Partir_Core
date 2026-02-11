package alpha

// CatalogSource represents an external data source for catalogs.
// Instead of embedding millions of records as JSON, CatalogSource points
// to external storage (Postgres, S3/MinIO, CSV URL, or API endpoints).
// Alpha resolves these at build time or caches them at ticket time.
//
// For API sources, Alpha maintains awareness of public APIs including:
//   - Government: data.gov, Census, SEC EDGAR, FDA, USDA
//   - Open datasets: OpenStreetMap, Wikipedia/Wikidata, UN Data
//   - Industry-specific: IGDB (games), TMDb (movies), ClinicalTrials.gov (medical)
type CatalogSource struct {
	// ID is the unique identifier for this catalog source
	ID string `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Type is the source type: "embedded", "postgres", "s3", "csv_url", "api"
	Type string `json:"type"`

	// URI is the connection string, URL, or API endpoint
	URI string `json:"uri"`

	// Query is the SQL query, S3 prefix, or API path to fetch data
	Query string `json:"query,omitempty"`

	// Auth is the authentication method for API sources: "none", "api_key", "oauth"
	Auth string `json:"auth,omitempty"`

	// APIKeyHeader is the header name for API key authentication (e.g., "X-API-Key")
	APIKeyHeader string `json:"api_key_header,omitempty"`

	// CacheFor controls how long resolved data is cached: "1h", "24h", "never"
	CacheFor string `json:"cache_for,omitempty"`

	// Schema describes the expected structure of records from this source
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// KnownAPISources is a curated list of public APIs that Alpha can suggest
// as catalog sources when relevant to a project.
var KnownAPISources = []CatalogSource{
	// Government
	{ID: "data_gov", Name: "Data.gov", Type: "api", URI: "https://catalog.data.gov/api/3", Auth: "none"},
	{ID: "census", Name: "US Census Bureau", Type: "api", URI: "https://api.census.gov/data", Auth: "api_key"},
	{ID: "sec_edgar", Name: "SEC EDGAR", Type: "api", URI: "https://efts.sec.gov/LATEST", Auth: "none"},
	{ID: "fda_openfda", Name: "FDA OpenFDA", Type: "api", URI: "https://api.fda.gov", Auth: "none"},

	// Open datasets
	{ID: "osm_nominatim", Name: "OpenStreetMap Nominatim", Type: "api", URI: "https://nominatim.openstreetmap.org", Auth: "none"},
	{ID: "wikidata", Name: "Wikidata", Type: "api", URI: "https://www.wikidata.org/w/api.php", Auth: "none"},

	// Industry-specific
	{ID: "igdb", Name: "IGDB (Games)", Type: "api", URI: "https://api.igdb.com/v4", Auth: "oauth"},
	{ID: "tmdb", Name: "TMDb (Movies)", Type: "api", URI: "https://api.themoviedb.org/3", Auth: "api_key", APIKeyHeader: "Authorization"},
	{ID: "pexels", Name: "Pexels (Stock Images)", Type: "api", URI: "https://api.pexels.com/v1", Auth: "api_key", APIKeyHeader: "Authorization"},
}

// FindRelevantAPIs searches the known API sources for ones relevant to the given keywords.
func FindRelevantAPIs(keywords []string) []CatalogSource {
	var matches []CatalogSource
	for _, src := range KnownAPISources {
		for _, kw := range keywords {
			name := src.Name + " " + src.ID
			if containsInsensitive(name, kw) {
				matches = append(matches, src)
				break
			}
		}
	}
	return matches
}

// containsInsensitive checks if s contains substr (case-insensitive).
func containsInsensitive(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			containsLower(toLower(s), toLower(substr)))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
