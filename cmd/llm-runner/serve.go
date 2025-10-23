package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/geppetto/cmd/llm-runner/templates"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/rs/zerolog/log"
)

type ServeSettings struct {
	Out  string `glazed.parameter:"out"`
	Port int    `glazed.parameter:"port"`
}

type ServeCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*ServeCommand)(nil)

func NewServeCommand() (*ServeCommand, error) {
	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Start web UI to visualize artifacts"),
		cmds.WithFlags(
			parameters.NewParameterDefinition("out", parameters.ParameterTypeString, parameters.WithDefault("out"), parameters.WithHelp("Artifacts directory")),
			parameters.NewParameterDefinition("port", parameters.ParameterTypeInteger, parameters.WithDefault(8080), parameters.WithHelp("HTTP port")),
		),
	)
	return &ServeCommand{CommandDescription: desc}, nil
}

func (c *ServeCommand) Run(ctx context.Context, parsed *layers.ParsedLayers) error {
	if err := logging.InitLoggerFromViper(); err != nil {
		return err
	}
	s := &ServeSettings{}
	if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	// API handlers for parsed data
	apiHandler := &APIHandler{BaseDir: s.Out}

	// Legacy handlers for direct file access
	legacyHandler := &ArtifactHandler{BaseDir: s.Out}

	mux := http.NewServeMux()

	// New API endpoints
	mux.HandleFunc("/api/runs", apiHandler.GetRunsHandler)
	mux.HandleFunc("/api/runs/", apiHandler.GetRunHandler)

	// Legacy endpoints
	mux.HandleFunc("/legacy/", legacyHandler.IndexHandler)
	mux.HandleFunc("/api/artifacts", legacyHandler.ArtifactsHandler)
	mux.HandleFunc("/api/file", legacyHandler.FileHandler)

	// Serve React frontend (will be built dist)
	// Try to find dist directory relative to binary location
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	distDirs := []string{
		filepath.Join(execDir, "web", "dist"),
		filepath.Join("cmd", "llm-runner", "web", "dist"),
		filepath.Join("web", "dist"),
	}

	var distDir string
	for _, dir := range distDirs {
		if _, err := os.Stat(dir); err == nil {
			distDir = dir
			break
		}
	}

	if distDir != "" {
		// SPA handler: serve index.html for all non-API, non-legacy routes
		spaHandler := &SPAHandler{
			StaticPath: distDir,
			IndexPath:  filepath.Join(distDir, "index.html"),
		}
		mux.Handle("/", spaHandler)
		log.Info().Str("dist", distDir).Msg("Serving React frontend")
	} else {
		// Fallback to legacy UI
		mux.HandleFunc("/", legacyHandler.IndexHandler)
		log.Warn().Msg("React dist not found, using legacy UI")
	}

	addr := fmt.Sprintf(":%d", s.Port)
	log.Info().Str("addr", addr).Str("base", s.Out).Msg("Starting web server")
	fmt.Printf("Server running at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

type ArtifactHandler struct {
	BaseDir string
}

func (h *ArtifactHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	groups, err := h.scanArtifacts()
	if err != nil {
		log.Error().Err(err).Msg("failed to scan artifacts")
		http.Error(w, "Failed to scan artifacts", http.StatusInternalServerError)
		return
	}
	templates.Index(groups, h.BaseDir).Render(r.Context(), w)
}

func (h *ArtifactHandler) ArtifactsHandler(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("dir")
	if dirPath == "" {
		dirPath = "."
	}
	fullPath := filepath.Join(h.BaseDir, dirPath)
	files, err := h.listFiles(fullPath)
	if err != nil {
		log.Error().Err(err).Str("path", fullPath).Msg("failed to list files")
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}
	templates.FileList(files, dirPath).Render(r.Context(), w)
}

func (h *ArtifactHandler) FileHandler(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// Security: prevent path traversal
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(h.BaseDir, cleanPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Error().Err(err).Str("path", fullPath).Msg("failed to read file")
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	templates.FileContent(string(content), filepath.Base(relPath)).Render(r.Context(), w)
}

func (h *ArtifactHandler) scanArtifacts() ([]templates.ArtifactGroup, error) {
	var groups []templates.ArtifactGroup

	// Check if base directory exists
	if _, err := os.Stat(h.BaseDir); os.IsNotExist(err) {
		return groups, nil
	}

	err := filepath.WalkDir(h.BaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		files, err := h.listFiles(path)
		if err != nil {
			return nil // Skip directories we can't read
		}

		if len(files) > 0 {
			relPath, _ := filepath.Rel(h.BaseDir, path)
			if relPath == "." {
				relPath = ""
			}
			groups = append(groups, templates.ArtifactGroup{
				DirPath: path,
				RelPath: relPath,
				Files:   files,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by relative path
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].RelPath == "" {
			return true
		}
		if groups[j].RelPath == "" {
			return false
		}
		return groups[i].RelPath < groups[j].RelPath
	})

	return groups, nil
}

func (h *ArtifactHandler) listFiles(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}

// SPAHandler serves a Single Page Application
// It serves static files normally but returns index.html for routes that don't exist
type SPAHandler struct {
	StaticPath string
	IndexPath  string
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute path to prevent directory traversal
	path := filepath.Join(h.StaticPath, r.URL.Path)

	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info.IsDir() {
		// File doesn't exist or is a directory, serve index.html
		http.ServeFile(w, r, h.IndexPath)
		return
	}

	if err != nil {
		// Some other error occurred
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serve the file
	http.FileServer(http.Dir(h.StaticPath)).ServeHTTP(w, r)
}
