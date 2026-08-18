package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := "."
	errors := []string{}

	// Gate A: Allowed path allowlist
	allowedPaths := []string{
		"cmd", "internal", "pkg", "migrations", "rulebooks", "References",
		"deploy", "examples", "go.mod", "go.sum", "Makefile", "README.md",
		"docker-compose.yml", ".git", ".gitignore", ".vscode", "bin",
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		// Skip recognized hidden dirs
		if strings.HasPrefix(d.Name(), ".") && d.IsDir() {
			return filepath.SkipDir
		}

		// Check top-level directories
		if filepath.Dir(path) == "." {
			allowed := false
			for _, p := range allowedPaths {
				if path == p {
					allowed = true
					break
				}
			}
			if !allowed {
				errors = append(errors, fmt.Sprintf("Gate A: Forbidden top-level path: %s", path))
				if d.IsDir() {
					return filepath.SkipDir
				}
			}
		}

		// Gate B: Domain denylist
		deniedTerms := []string{"game", "item", "npc", "quest", "render_2d"}
		// Allow examples/ and References/ to contain these terms
		if !strings.HasPrefix(path, "examples") && !strings.HasPrefix(path, "References") {
			for _, term := range deniedTerms {
				if strings.Contains(strings.ToLower(d.Name()), term) {
					errors = append(errors, fmt.Sprintf("Gate B: Forbidden domain term %q in path: %s", term, path))
				}
			}
		}

		return nil
	})
	if err != nil {
		fmt.Printf("Error walking repo: %v\n", err)
		os.Exit(1)
	}

	// Gate C: Imports boundary
	// Disallow importing plugins/ (now examples/plugins/) or domain roots from internal/ or cmd/
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Only check internal/ and cmd/
		if !strings.HasPrefix(path, "internal") && !strings.HasPrefix(path, "cmd") {
			return nil
		}

		// Parse file imports
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil // Skip parse errors
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")

			// Check for forbidden imports
			// 1. examples/plugins
			// 2. domain roots (if any existed, but Gate A/B blocked them)
			// effectively we just block importing "partir/core/examples" or similar

			if strings.Contains(importPath, "examples/plugins") {
				errors = append(errors, fmt.Sprintf("Gate C: Forbidden import %q in %s", importPath, path))
			}
		}
		return nil
	})

	// Gate D: No parallel implementations
	// Check for patterns that indicate duplicate subsystems
	parallelPatterns := map[string]string{
		"metrics_server":   "Use pkg/telemetry OTel exporter instead of standalone metrics server",
		"telemetry_server": "Use pkg/telemetry OTel exporter instead of standalone server",
		"prom_server":      "Use pkg/telemetry OTel exporter instead of Prometheus server",
		"custom_metrics":   "Use pkg/telemetry for metrics, not custom implementation",
		"prometheus.go":    "Use pkg/telemetry for metrics, extends existing OTel setup",
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Skip pkg/metrics (legacy, being replaced by telemetry)
		if strings.HasPrefix(path, "pkg"+string(os.PathSeparator)+"metrics") {
			return nil
		}

		name := strings.ToLower(d.Name())
		for pattern, reason := range parallelPatterns {
			if strings.Contains(name, pattern) {
				errors = append(errors, fmt.Sprintf("Gate D: Parallel implementation detected %q in %s - %s", pattern, path, reason))
			}
		}
		return nil
	})

	// Gate D (continued): Check go.mod for dependency conflicts
	goModPath := filepath.Join(root, "go.mod")
	if goModContent, err := os.ReadFile(goModPath); err == nil {
		modStr := string(goModContent)
		hasOtel := strings.Contains(modStr, "go.opentelemetry.io/otel")
		hasPromClient := strings.Contains(modStr, "prometheus/client_golang")

		// If both OTel and Prometheus client exist, check for prometheus server patterns
		if hasOtel && hasPromClient {
			// Scan for promhttp.Handler() usage outside of OTel context
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				// Skip pkg/metrics (legacy) and cmd/guard (self)
				normalPath := filepath.ToSlash(path)
				if strings.HasPrefix(normalPath, "pkg/metrics") || strings.HasPrefix(normalPath, "cmd/guard") {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				contentStr := string(content)

				// Check for standalone Prometheus HTTP handlers
				if strings.Contains(contentStr, "promhttp.Handler()") {
					errors = append(errors, fmt.Sprintf("Gate D: promhttp.Handler() found in %s - use OTel exporter instead", path))
				}

				// Check for http.ListenAndServe in metrics-context files
				if strings.Contains(contentStr, "http.ListenAndServe") || strings.Contains(contentStr, "http.Server{") {
					// Only flag if file looks metrics-related
					if strings.Contains(strings.ToLower(path), "metric") || strings.Contains(strings.ToLower(path), "prom") {
						errors = append(errors, fmt.Sprintf("Gate D: New HTTP server in metrics file %s - use shared service endpoint", path))
					}
				}

				// Check for prometheus.NewRegistry() outside approved locations
				if strings.Contains(contentStr, "prometheus.NewRegistry()") {
					errors = append(errors, fmt.Sprintf("Gate D: prometheus.NewRegistry() in %s - use pkg/telemetry OTel metrics", path))
				}

				return nil
			})
		}
	}

	// Gate E: Shadow OTel detection
	// Prevent creating parallel OTel setups outside pkg/telemetry
	shadowOtelPatterns := []string{
		"otel.SetMeterProvider(",
		"otel.SetTracerProvider(",
		"sdkmetric.NewMeterProvider(",
		"sdktrace.NewTracerProvider(",
		"otlptrace.New(",
		"otlpmetric.New(",
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Only pkg/telemetry is allowed to set up OTel providers
		// Also skip cmd/guard (contains string literals for detection)
		normalPath := filepath.ToSlash(path)
		if strings.HasPrefix(normalPath, "pkg/telemetry") || strings.HasPrefix(normalPath, "cmd/guard") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)

		for _, pattern := range shadowOtelPatterns {
			if strings.Contains(contentStr, pattern) {
				errors = append(errors, fmt.Sprintf("Gate E: Shadow OTel setup %q in %s - only pkg/telemetry may configure OTel providers", pattern, path))
			}
		}
		return nil
	})

	// Gate F: Diff budget (when running in CI with GUARD_DIFF_MODE=1)
	if os.Getenv("GUARD_DIFF_MODE") == "1" {
		maxNewFiles := 15
		maxNewPackages := 3

		newFileCount := 0
		newPackages := make(map[string]bool)

		// Read allowed_new_files from ticket if exists
		allowedFiles := make(map[string]bool)
		if ticketDir := os.Getenv("GUARD_TICKET_DIR"); ticketDir != "" {
			// Would read allowed_new_files from ticket plan here
			// For now, just count new files
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			// In diff mode, we'd compare against git baseline
			// For static analysis, we count total .go files as proxy
			normalPath := filepath.ToSlash(path)

			// Track packages
			dir := filepath.Dir(normalPath)
			if !strings.HasPrefix(dir, "examples") && !strings.HasPrefix(dir, ".") {
				newPackages[dir] = true
			}

			// Check if file is in allowed list
			if !allowedFiles[normalPath] {
				newFileCount++
			}
			return nil
		})

		if len(newPackages) > maxNewPackages {
			// Only warn in diff mode, not fail (packages accumulate over time)
			fmt.Printf("Gate F Warning: %d packages exceed recommended max of %d\n", len(newPackages), maxNewPackages)
		}
		if newFileCount > maxNewFiles {
			errors = append(errors, fmt.Sprintf("Gate F: %d new files exceed max of %d - decompose into smaller changes", newFileCount, maxNewFiles))
		}
	}

	if len(errors) > 0 {
		fmt.Println("Architectural Guard Failed:")
		for _, e := range errors {
			fmt.Printf("  %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Architectural Guard Passed")
}
