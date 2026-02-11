package doctor

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CheckResult represents a single diagnostic check result
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", "fail"
	Message string `json:"message"`
}

// Report contains all diagnostic results
type Report struct {
	Checks    []CheckResult `json:"checks"`
	GoVersion string        `json:"go_version"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
}

// RunDiagnostics performs all environment checks
func RunDiagnostics() *Report {
	report := &Report{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	report.Checks = append(report.Checks, checkEnvVar("PARTIR_DB_URL", true))
	report.Checks = append(report.Checks, checkEnvVar("PARTIR_JWT_SECRET", true))
	report.Checks = append(report.Checks, checkEnvVar("NATS_URL", false))
	report.Checks = append(report.Checks, checkEnvVar("PARTIR_MINIO_ENDPOINT", true))
	report.Checks = append(report.Checks, checkPostgres())
	report.Checks = append(report.Checks, checkNATS())
	report.Checks = append(report.Checks, checkTool("docker", "--version"))
	report.Checks = append(report.Checks, checkTool("mc", "--version"))
	report.Checks = append(report.Checks, checkDiskSpace())

	return report
}

// Print outputs the report with colors
func (r *Report) Print() {
	fmt.Printf("\n🩺 Partir Doctor — Environment Diagnostics\n")
	fmt.Printf("   Go: %s  |  OS: %s/%s\n\n", r.GoVersion, r.OS, r.Arch)

	for _, c := range r.Checks {
		icon := "✅"
		if c.Status == "warn" {
			icon = "⚠️ "
		} else if c.Status == "fail" {
			icon = "❌"
		}
		fmt.Printf("  %s  %-25s %s\n", icon, c.Name, c.Message)
	}
	fmt.Println()
}

func checkEnvVar(name string, required bool) CheckResult {
	val := os.Getenv(name)
	if val == "" {
		if required {
			return CheckResult{Name: name, Status: "fail", Message: "not set (required)"}
		}
		return CheckResult{Name: name, Status: "warn", Message: "not set (optional)"}
	}
	// Mask sensitive values
	masked := val[:min(4, len(val))] + strings.Repeat("*", max(0, len(val)-4))
	return CheckResult{Name: name, Status: "ok", Message: masked}
}

func checkPostgres() CheckResult {
	dbURL := os.Getenv("PARTIR_DB_URL")
	if dbURL == "" {
		return CheckResult{Name: "PostgreSQL", Status: "fail", Message: "PARTIR_DB_URL not set"}
	}

	u, err := url.Parse(dbURL)
	if err != nil {
		return CheckResult{Name: "PostgreSQL", Status: "fail", Message: "invalid URL"}
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}

	return checkTCPConnection("PostgreSQL", host, port)
}

func checkNATS() CheckResult {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	u, err := url.Parse(natsURL)
	if err != nil {
		return CheckResult{Name: "NATS", Status: "warn", Message: "invalid URL"}
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "4222"
	}

	return checkTCPConnection("NATS", host, port)
}

func checkTCPConnection(name, host, port string) CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return CheckResult{Name: name, Status: "fail", Message: fmt.Sprintf("unreachable at %s:%s", host, port)}
	}
	conn.Close()
	return CheckResult{Name: name, Status: "ok", Message: fmt.Sprintf("reachable at %s:%s", host, port)}
}

func checkTool(name, versionFlag string) CheckResult {
	cmd := exec.Command(name, versionFlag)
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Name: name, Status: "warn", Message: "not found in PATH"}
	}
	version := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if len(version) > 60 {
		version = version[:60] + "..."
	}
	return CheckResult{Name: name, Status: "ok", Message: version}
}

func checkDiskSpace() CheckResult {
	// Simple check — just verify we can write to current dir
	f, err := os.CreateTemp(".", ".partir-doctor-*")
	if err != nil {
		return CheckResult{Name: "Disk Write", Status: "fail", Message: "cannot write to current directory"}
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return CheckResult{Name: "Disk Write", Status: "ok", Message: "current directory is writable"}
}

// Note: Uses Go 1.21+ builtin min() and max() functions
