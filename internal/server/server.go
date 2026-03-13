package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/repomap/repomap/internal/renderer"
)

// Server serves the report frontend and API endpoints.
type Server struct {
	data     *renderer.ReportData
	addr     string
	server   *http.Server
	mu       sync.RWMutex
	bundleJS string
	cssData  string
}

// New creates a new Server instance.
func New(data *renderer.ReportData, addr string) *Server {
	return &Server{
		data:     data,
		addr:     addr,
		bundleJS: renderer.BundleJS,
		cssData:  renderer.StylesCSS,
	}
}

// Start begins serving and returns the URL. It blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/report", s.handleReport)
	mux.HandleFunc("/api/graph", s.handleGraph)
	mux.HandleFunc("/api/architecture", s.handleArchitecture)
	mux.HandleFunc("/api/modules/", s.handleModule)

	// Frontend
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/assets/bundle.js", s.handleBundle)
	mux.HandleFunc("/assets/styles.css", s.handleStyles)

	s.server = &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.addr, err)
	}

	url := fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	fmt.Printf("\n  ✦  Opening at %s\n", url)
	openBrowser(url)

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serving: %w", err)
	}
	return nil
}

// handleReport returns the full report data as JSON.
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.data)
}

// handleGraph returns the dependency graph.
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.data.Graph)
}

// handleArchitecture returns the architecture synthesis.
func (s *Server) handleArchitecture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.Architecture == nil {
		http.Error(w, "no architecture synthesis available", http.StatusNotFound)
		return
	}
	writeJSON(w, s.data.Architecture)
}

// handleModule returns a specific module's summary by path.
func (s *Server) handleModule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/modules/")
	if path == "" {
		http.Error(w, "module path required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, summary := range s.data.Summaries {
		if summary.FilePath == path {
			writeJSON(w, summary)
			return
		}
	}

	http.Error(w, "module not found", http.StatusNotFound)
}

// handleIndex serves the HTML shell that loads the frontend app.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		// SPA fallback: serve index for any non-asset route
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/assets/") {
			s.serveIndex(w)
			return
		}
		http.NotFound(w, r)
		return
	}
	s.serveIndex(w)
}

func (s *Server) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "Repomap"
	if s.data.Metadata != nil {
		title = s.data.Metadata.Name + " — Repomap"
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <link rel="stylesheet" href="/assets/styles.css">
</head>
<body>
  <div id="app"></div>
  <script src="/assets/bundle.js"></script>
</body>
</html>`, title)
}

// handleBundle serves the compiled JavaScript bundle.
func (s *Server) handleBundle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, s.bundleJS)
}

// handleStyles serves the compiled CSS.
func (s *Server) handleStyles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, s.cssData)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("encoding response: %v", err), http.StatusInternalServerError)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
