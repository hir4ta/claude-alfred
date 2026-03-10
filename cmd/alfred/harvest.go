package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/install"
	"github.com/hir4ta/claude-alfred/internal/store"
)

// clearLine moves cursor to column 0 and erases the entire line.
const clearLine = "\r\033[2K"

// runHarvest performs a live crawl of documentation sources and updates the
// knowledge base. This is the user-facing equivalent of the background
// crawl-async subcommand, with interactive progress output.
func runHarvest() error {
	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Clean up expired docs first.
	if n, err := st.DeleteExpiredDocs(ctx); err == nil && n > 0 {
		fmt.Printf("  Cleaned %d expired docs\n", n)
	}

	// Load custom sources.
	var customSources []install.CustomSource
	cwd, _ := os.Getwd()
	if cfg := loadProjectConfig(cwd); cfg != nil {
		for _, cs := range cfg.CustomSources {
			customSources = append(customSources, install.CustomSource{URL: cs.URL, Label: cs.Label})
		}
	}
	if globalSources := loadGlobalCustomSources(); len(globalSources) > 0 {
		customSources = append(customSources, globalSources...)
	}

	fmt.Println("  Crawling documentation sources...")
	start := time.Now()

	// phaseProgress returns a callback that clears the line before printing,
	// and prints a newline when the phase completes (done == total).
	phaseProgress := func(label string) func(done, total int) {
		return func(done, total int) {
			fmt.Printf("%s  %s [%d/%d]", clearLine, label, done, total)
			if done == total {
				fmt.Println()
			}
		}
	}

	sf, crawlStats, err := install.Crawl(ctx, &install.CrawlProgress{
		OnDocsPage:   phaseProgress("Docs"),
		OnBlogPost:   phaseProgress("Engineering blog"),
		OnClaudeBlog: phaseProgress("Claude blog"),
		OnNews:       phaseProgress("News"),
		OnAgentSDK:   phaseProgress("Agent SDK"),
	}, st, customSources)
	if sf == nil {
		return fmt.Errorf("crawl failed: %w", err)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: %v\n", err)
	}

	if crawlStats != nil {
		fmt.Printf("  Fetched %d, skipped %d (not modified)\n", crawlStats.Fetched, crawlStats.NotModified)
	}

	// Embedder is optional.
	var emb *embedder.Embedder
	if e, err := embedder.NewEmbedder(); err == nil {
		emb = e
		st.ExpectedDims = e.Dims()
	}

	res, err := install.ApplySeedData(ctx, st, emb, sf, &install.SeedProgress{
		OnDocUpsert:  phaseProgress("Upserting docs"),
		OnEmbedBatch: phaseProgress("Embedding"),
	})
	if err != nil {
		return fmt.Errorf("apply seed data: %w", err)
	}

	mode := "FTS-only"
	if emb != nil {
		mode = fmt.Sprintf("%d embeddings", res.Embedded)
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("\n  ✓ Harvest complete (%s)\n", elapsed)
	fmt.Printf("  %d applied, %d unchanged (%s)\n", res.Applied, res.Unchanged, mode)

	return nil
}
