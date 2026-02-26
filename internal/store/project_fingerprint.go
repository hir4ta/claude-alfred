package store

import (
	"os"
	"path/filepath"
	"strings"
)

// GenerateFingerprint detects project characteristics from the directory structure.
func GenerateFingerprint(projectPath string) *ProjectFingerprint {
	fp := &ProjectFingerprint{
		ProjectPath: projectPath,
		ProjectName: filepath.Base(projectPath),
	}

	// Detect languages and frameworks from manifest files.
	if hasFile(projectPath, "go.mod") {
		fp.Languages = append(fp.Languages, "go")
		fp.Frameworks = append(fp.Frameworks, detectGoFrameworks(projectPath)...)
	}
	if hasFile(projectPath, "package.json") {
		fp.Languages = append(fp.Languages, "js")
		fp.Frameworks = append(fp.Frameworks, detectJSFrameworks(projectPath)...)
	}
	if hasFile(projectPath, "requirements.txt") || hasFile(projectPath, "pyproject.toml") || hasFile(projectPath, "setup.py") {
		fp.Languages = append(fp.Languages, "python")
	}
	if hasFile(projectPath, "Cargo.toml") {
		fp.Languages = append(fp.Languages, "rust")
	}
	if hasFile(projectPath, "Gemfile") {
		fp.Languages = append(fp.Languages, "ruby")
	}

	// Detect domains from directory names.
	fp.Domains = detectDomains(projectPath)

	return fp
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func detectGoFrameworks(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil
	}
	content := string(data)
	var fws []string
	frameworkPaths := map[string]string{
		"github.com/gin-gonic/gin":   "gin",
		"github.com/labstack/echo":   "echo",
		"github.com/gofiber/fiber":   "fiber",
		"github.com/gorilla/mux":     "gorilla",
		"github.com/go-chi/chi":      "chi",
		"gorm.io/gorm":               "gorm",
		"github.com/charmbracelet/bubbletea": "bubbletea",
		"github.com/mark3labs/mcp-go": "mcp-go",
	}
	for path, name := range frameworkPaths {
		if strings.Contains(content, path) {
			fws = append(fws, name)
		}
	}
	return fws
}

func detectJSFrameworks(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	content := string(data)
	var fws []string
	frameworkNames := map[string]string{
		"\"react\"":      "react",
		"\"next\"":       "next",
		"\"vue\"":        "vue",
		"\"express\"":    "express",
		"\"fastify\"":    "fastify",
		"\"svelte\"":     "svelte",
		"\"typescript\"": "typescript",
	}
	for key, name := range frameworkNames {
		if strings.Contains(content, key) {
			fws = append(fws, name)
		}
	}
	return fws
}

func detectDomains(dir string) []string {
	var domains []string
	domainDirs := map[string]string{
		"auth":     "auth",
		"api":      "api",
		"database": "database",
		"db":       "database",
		"ui":       "ui",
		"frontend": "ui",
		"web":      "ui",
		"infra":    "infra",
		"deploy":   "infra",
		"config":   "config",
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if domain, ok := domainDirs[name]; ok && !seen[domain] {
			seen[domain] = true
			domains = append(domains, domain)
		}
	}
	return domains
}
