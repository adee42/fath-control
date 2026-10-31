// Command mcpb builds the Claude Desktop extension bundle.
//
// It cross-compiles the MCP server for every supported platform and packs
// the binaries with extension/manifest.json into bin/fath-control-<version>.mcpb
// (a zip with manifest.json at its root), plus bin/SHA256SUMS.
//
// Run from the repository root: go run ./scripts/mcpb
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type target struct {
	goos, goarch string
	file         string // name inside the bundle's server/ directory
}

// Keep in sync with server.mcp_config.platform_overrides in extension/manifest.json
// (checked below).
var targets = []target{
	{"windows", "amd64", "fath-mcp-server.exe"},
	{"darwin", "arm64", "fath-mcp-server-darwin-arm64"},
	{"linux", "amd64", "fath-mcp-server-linux-amd64"},
}

type bundleFile struct {
	name string // path inside the zip
	path string // path on disk
	mode os.FileMode
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run from the repository root (go.mod not found)")
	}

	manifestPath := filepath.Join("extension", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		Version string `json:"version"`
		Server  struct {
			MCPConfig struct {
				PlatformOverrides map[string]struct {
					Command string `json:"command"`
				} `json:"platform_overrides"`
			} `json:"mcp_config"`
		} `json:"server"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("parsing %s: %w", manifestPath, err)
	}
	if manifest.Version == "" {
		return fmt.Errorf("%s has no version", manifestPath)
	}

	stage, err := os.MkdirTemp("", "mcpb-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	files := []bundleFile{{name: "manifest.json", path: manifestPath, mode: 0o644}}
	for _, t := range targets {
		out := filepath.Join(stage, t.file)
		// -trimpath drops local build paths, -buildvcs=false drops VCS metadata,
		// -s -w strips symbol and debug tables.
		cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w", "-o", out, "./cmd/fath-mcp-server")
		cmd.Env = append(os.Environ(), "GOOS="+t.goos, "GOARCH="+t.goarch, "CGO_ENABLED=0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Printf("building %s/%s -> server/%s\n", t.goos, t.goarch, t.file)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go build %s/%s: %w", t.goos, t.goarch, err)
		}
		files = append(files, bundleFile{name: "server/" + t.file, path: out, mode: 0o755})
	}

	// Every command the manifest can launch must exist in the bundle.
	inBundle := map[string]bool{}
	for _, f := range files {
		inBundle[f.name] = true
	}
	for platform, o := range manifest.Server.MCPConfig.PlatformOverrides {
		rel := strings.TrimPrefix(o.Command, "${__dirname}/")
		if !inBundle[rel] {
			return fmt.Errorf("manifest platform_overrides.%s.command %q is not in the bundle", platform, o.Command)
		}
	}

	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}
	mcpb := filepath.Join("bin", fmt.Sprintf("fath-control-%s.mcpb", manifest.Version))
	sum, err := writeZip(mcpb, files)
	if err != nil {
		return err
	}
	sums := fmt.Sprintf("%s  %s\n", sum, filepath.Base(mcpb))
	if err := os.WriteFile(filepath.Join("bin", "SHA256SUMS"), []byte(sums), 0o644); err != nil {
		return err
	}
	fmt.Printf("built %s\nsha256 %s\n", mcpb, sum)
	return nil
}

// writeZip writes files to a zip at path and returns its SHA-256. Entry
// timestamps are fixed and unix modes are stored (so the Linux/macOS binaries
// stay executable), which also makes the output reproducible.
func writeZip(path string, files []bundleFile) (string, error) {
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(f, h))
	modified := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, bf := range files {
		hdr := &zip.FileHeader{Name: bf.name, Method: zip.Deflate, Modified: modified}
		hdr.SetMode(bf.mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return "", err
		}
		src, err := os.Open(bf.path)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(w, src)
		src.Close()
		if err != nil {
			return "", err
		}
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
