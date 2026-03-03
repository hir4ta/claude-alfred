package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/install"
	"github.com/hir4ta/claude-alfred/internal/mcpserver"
	"github.com/hir4ta/claude-alfred/internal/store"
)

// Build info set at build time via ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve":
		return runServe()
	case "setup":
		return runSetup()
	case "harvest":
		return runHarvest()
	case "crawl-seed":
		output := "internal/install/seed_docs.json"
		if len(os.Args) > 2 {
			output = os.Args[2]
		}
		return install.CrawlSeed(output)
	case "plugin-bundle":
		outputDir := "./plugin"
		if len(os.Args) > 2 {
			outputDir = os.Args[2]
		}
		return install.Bundle(outputDir, version)
	case "hook":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: alfred hook <EventName>")
		}
		return runHook(os.Args[2])
	case "version", "--version", "-v":
		printVersion()
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		if cmd == "" {
			return nil
		}
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func runServe() error {
	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}
	defer st.Close()

	emb, err := embedder.NewEmbedder()
	if err != nil {
		return fmt.Errorf("VOYAGE_API_KEY is required: %w", err)
	}

	if count, _ := st.SeedDocsCount(); count == 0 {
		fmt.Fprintln(os.Stderr, "Warning: no seed docs found. Run 'alfred setup' to initialize.")
	}

	s := mcpserver.New(st, emb)
	return server.ServeStdio(s)
}

func printVersion() {
	c, d := commit, date
	if c == "unknown" {
		// Fallback for dev builds: read VCS info from Go build info.
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if len(s.Value) > 7 {
						c = s.Value[:7]
					} else {
						c = s.Value
					}
				case "vcs.time":
					d = s.Value
				}
			}
		}
	}
	if c != "unknown" {
		fmt.Printf("alfred %s (%s %s)\n", version, c, d)
	} else {
		fmt.Printf("alfred %s\n", version)
	}
}

func printUsage() {
	fmt.Println(`alfred - Your silent butler for Claude Code

Usage:
  alfred [command]

Commands:
  serve          Run as MCP server (stdio) for Claude Code integration
  setup          Initialize knowledge base (seed docs + generate embeddings)
  harvest        Refresh knowledge base (crawl + embed fresh docs)
  version        Show version
  help           Show this help

Environment:
  VOYAGE_API_KEY     Required. Enables semantic vector search with Voyage AI.`)
}
