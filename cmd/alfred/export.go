package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hir4ta/claude-alfred/internal/store"
)

type exportDoc struct {
	URL         string `json:"url"`
	SectionPath string `json:"section_path"`
	Content     string `json:"content"`
	SourceType  string `json:"source_type"`
	CrawledAt   string `json:"crawled_at"`
}

type exportData struct {
	ExportedAt string      `json:"exported_at"`
	Version    string      `json:"version"`
	Memories   []exportDoc `json:"memories"`
	Specs      []exportDoc `json:"specs,omitempty"`
}

func runExport() error {
	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ctx := context.Background()
	data := exportData{
		ExportedAt: time.Now().Format(time.RFC3339),
		Version:    resolvedVersion(),
	}

	// Export memories.
	rows, err := st.DB().QueryContext(ctx,
		`SELECT url, section_path, content, source_type, crawled_at
		 FROM docs WHERE source_type = 'memory' ORDER BY crawled_at DESC`)
	if err != nil {
		return fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d exportDoc
		if err := rows.Scan(&d.URL, &d.SectionPath, &d.Content, &d.SourceType, &d.CrawledAt); err != nil {
			debugf("export: scan memory: %v", err)
			continue
		}
		data.Memories = append(data.Memories, d)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate memories: %w", err)
	}

	// Export specs (if --all flag).
	for _, arg := range os.Args[2:] {
		if arg == "--all" {
			specRows, err := st.DB().QueryContext(ctx,
				`SELECT url, section_path, content, source_type, crawled_at
				 FROM docs WHERE source_type = 'spec' ORDER BY url`)
			if err != nil {
				return fmt.Errorf("query specs: %w", err)
			}
			for specRows.Next() {
				var d exportDoc
				if err := specRows.Scan(&d.URL, &d.SectionPath, &d.Content, &d.SourceType, &d.CrawledAt); err != nil {
					debugf("export: scan spec: %v", err)
					continue
				}
				data.Specs = append(data.Specs, d)
			}
			if err := specRows.Err(); err != nil {
				specRows.Close()
				return fmt.Errorf("iterate specs: %w", err)
			}
			specRows.Close()
			break
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Exported %d memories", len(data.Memories))
	if len(data.Specs) > 0 {
		fmt.Fprintf(os.Stderr, ", %d specs", len(data.Specs))
	}
	fmt.Fprintln(os.Stderr)
	return nil
}
