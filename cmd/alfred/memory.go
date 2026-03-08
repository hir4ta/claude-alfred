package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/store"
)

// defaultMemoryMaxAgeDays is the default maximum age for memory pruning.
const defaultMemoryMaxAgeDays = 180

func memoryMaxAgeDays() int {
	if v := os.Getenv("ALFRED_MEMORY_MAX_AGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMemoryMaxAgeDays
}

func runMemory() error {
	if len(os.Args) < 3 {
		fmt.Println(`alfred memory — manage persistent memories

Commands:
  prune [--confirm]    Remove old memories (default: dry-run preview)
  stats                Show memory statistics

Options:
  --max-age DAYS       Maximum age in days (default: 180, env: ALFRED_MEMORY_MAX_AGE_DAYS)`)
		return nil
	}

	switch os.Args[2] {
	case "prune":
		return runMemoryPrune()
	case "stats":
		return runMemoryStats()
	default:
		return fmt.Errorf("unknown memory command: %s", os.Args[2])
	}
}

func runMemoryPrune() error {
	confirm := slices.Contains(os.Args[2:], "--confirm")
	maxAge := memoryMaxAgeDays()

	// Parse --max-age flag.
	for i := 2; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--max-age" {
			if n, err := strconv.Atoi(os.Args[i+1]); err == nil && n > 0 {
				maxAge = n
			}
		}
	}

	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ctx := context.Background()
	cutoff := time.Now().AddDate(0, 0, -maxAge).Format(time.RFC3339)

	// Count candidates.
	var count int
	err = st.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM docs WHERE source_type = 'memory' AND crawled_at < ?`, cutoff).Scan(&count)
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}

	if count == 0 {
		fmt.Printf("No memories older than %d days.\n", maxAge)
		return nil
	}

	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	if !confirm {
		// Dry-run: show what would be deleted.
		fmt.Printf("Found %s older than %d days:\n\n",
			warnStyle.Render(fmt.Sprintf("%d memories", count)), maxAge)

		rows, err := st.DB().QueryContext(ctx,
			`SELECT section_path, crawled_at FROM docs
			 WHERE source_type = 'memory' AND crawled_at < ?
			 ORDER BY crawled_at ASC LIMIT 20`, cutoff)
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var path, crawled string
			if rows.Scan(&path, &crawled) == nil {
				dateStr := crawled
				if len(crawled) >= 10 {
					dateStr = crawled[:10]
				}
				fmt.Printf("  %s  %s\n", mutedStyle.Render(dateStr), path)
			}
		}
		if count > 20 {
			fmt.Printf("  %s\n", mutedStyle.Render(fmt.Sprintf("... and %d more", count-20)))
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate: %w", err)
		}
		fmt.Printf("\nRun with --confirm to delete. Consider 'alfred export' first.\n")
		return nil
	}

	// Actually delete.
	res, err := st.DB().ExecContext(ctx,
		`DELETE FROM docs WHERE source_type = 'memory' AND crawled_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	deleted, _ := res.RowsAffected()
	fmt.Printf("Deleted %d memories older than %d days.\n", deleted, maxAge)
	return nil
}

func runMemoryStats() error {
	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ctx := context.Background()

	var total int64
	if err := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM docs WHERE source_type = 'memory'`).Scan(&total); err != nil {
		return fmt.Errorf("count: %w", err)
	}

	fmt.Printf("Total memories: %d\n", total)

	rows, err := st.DB().QueryContext(ctx,
		`SELECT SUBSTR(section_path, 1, INSTR(section_path, ' > ')-1) AS project, COUNT(*) AS cnt,
		        MIN(crawled_at) AS oldest, MAX(crawled_at) AS newest
		 FROM docs WHERE source_type = 'memory'
		 GROUP BY project ORDER BY cnt DESC`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fmt.Println()
	for rows.Next() {
		var proj string
		var cnt int
		var oldest, newest string
		if rows.Scan(&proj, &cnt, &oldest, &newest) == nil && proj != "" {
			oldDate := oldest
			if len(oldest) >= 10 {
				oldDate = oldest[:10]
			}
			newDate := newest
			if len(newest) >= 10 {
				newDate = newest[:10]
			}
			fmt.Printf("  %-30s %3d memories  (%s — %s)\n", proj, cnt, oldDate, newDate)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	return nil
}
