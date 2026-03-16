package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/hir4ta/claude-alfred/internal/api"
	"github.com/hir4ta/claude-alfred/internal/dashboard"
	"github.com/hir4ta/claude-alfred/internal/embedder"
)

// webAssets will hold embedded SPA files once the build pipeline copies
// web/dist/ into cmd/alfred/web/dist/ before go build.
// For ALFRED_DEV=1 mode, this is unused (dev proxy to Vite).
//
//go:embed all:web/dist
var webAssets embed.FS

func runDashboard(port int, urlOnly bool) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Open store + embedder (same pattern as runServe in main.go).
	st, _ := openStore()
	if st != nil {
		defer st.Close()
	}
	var emb *embedder.Embedder
	if os.Getenv("VOYAGE_API_KEY") != "" {
		if e, err := embedder.NewEmbedder(); err == nil {
			emb = e
			if st != nil {
				st.ExpectedDims = emb.Dims()
			}
		}
	}
	ds := dashboard.NewFileDataSource(projectPath, st, emb)

	// Build API server options.
	var opts []api.Option
	opts = append(opts, api.WithVersion(resolvedVersion()))
	if os.Getenv("ALFRED_DEV") == "1" {
		opts = append(opts, api.WithDevProxy("http://localhost:5173"))
	} else {
		webFS, err := fs.Sub(webAssets, "web/dist")
		if err != nil {
			return fmt.Errorf("embedded web assets: %w", err)
		}
		opts = append(opts, api.WithEmbedFS(webFS))
	}

	specDir := projectPath + "/.alfred/specs"
	srv := api.New(ds, specDir, opts...)

	// Bind port (avoids TOCTOU race vs ListenAndServe).
	addr := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use", port)
	}

	// Start server in background.
	go func() {
		if err := srv.Serve(ln); err != nil && err.Error() != "http: Server closed" {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	url := fmt.Sprintf("http://%s", addr)
	if urlOnly {
		fmt.Println(url)
	} else {
		fmt.Fprintf(os.Stderr, "alfred dashboard: %s\n", url)
		openBrowser(url)
	}

	// Wait for signal, graceful shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Fprintln(os.Stderr, "\nshutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// openBrowser attempts to open the URL in the default browser.
// On failure, it silently returns (URL is already printed to stderr).
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start() //nolint:errcheck
}
