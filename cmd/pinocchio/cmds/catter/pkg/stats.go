package pkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/go-go-golems/clay/pkg/filefilter"

	"github.com/go-go-golems/clay/pkg/filewalker"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/weaviate/tiktoken-go"
)

type FileStats struct {
	TokenCount int
	LineCount  int
	Size       int64
	FileCount  int // Add FileCount field
}

type Stats struct {
	Files     map[string]FileStats
	FileTypes map[string]FileStats
	Dirs      map[string]FileStats
	DirFiles  map[string][]string // New field to store files per directory
	Total     FileStats
	mu        sync.Mutex
}

type OutputFlag int

const (
	OutputOverview OutputFlag = 1 << iota
	OutputDirStructure
	OutputFullStructure
)

type Config struct {
	OutputFlags OutputFlag
}

func NewStats() *Stats {
	return &Stats{
		Files:     make(map[string]FileStats),
		FileTypes: make(map[string]FileStats),
		Dirs:      make(map[string]FileStats),
		DirFiles:  make(map[string][]string), // Initialize the new map
	}
}

func (s *Stats) AddFile(path string, stats FileStats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Files[path] = stats

	// Update filetype stats
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = "no_extension"
	}
	fileTypeStats := s.FileTypes[ext]
	fileTypeStats.TokenCount += stats.TokenCount
	fileTypeStats.LineCount += stats.LineCount
	fileTypeStats.Size += stats.Size
	fileTypeStats.FileCount++ // Increment FileCount for file type
	s.FileTypes[ext] = fileTypeStats

	// Update directory stats
	dir := filepath.Dir(path)
	dirStats := s.Dirs[dir]
	dirStats.TokenCount += stats.TokenCount
	dirStats.LineCount += stats.LineCount
	dirStats.Size += stats.Size
	dirStats.FileCount++ // Increment FileCount for directory
	s.Dirs[dir] = dirStats

	// Update DirFiles map
	s.DirFiles[dir] = append(s.DirFiles[dir], path)

	// Update total stats
	s.Total.TokenCount += stats.TokenCount
	s.Total.LineCount += stats.LineCount
	s.Total.Size += stats.Size
	s.Total.FileCount++ // Increment FileCount for total
}

func (s *Stats) ComputeStats(paths []string, filter *filefilter.FileFilter) error {
	tokenCounter, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return fmt.Errorf("error initializing tiktoken: %v", err)
	}

	walker, err := filewalker.NewWalker(
		filewalker.WithPaths(paths),
		filewalker.WithFilter(filter.FilterNode),
	)
	if err != nil {
		return fmt.Errorf("error creating filewalker: %v", err)
	}

	preVisit := func(w *filewalker.Walker, node *filewalker.Node) error {
		if node.Type == filewalker.FileNode {
			content, err := os.ReadFile(node.Path)
			if err != nil {
				return fmt.Errorf("error reading file %s: %v", node.Path, err)
			}

			tokens := tokenCounter.Encode(string(content), nil, nil)
			tokenCount := len(tokens)
			lineCount := strings.Count(string(content), "\n") + 1
			size := int64(len(content))

			fileStats := FileStats{
				TokenCount: tokenCount,
				LineCount:  lineCount,
				Size:       size,
				FileCount:  1,
			}

			s.AddFile(node.Path, fileStats)
		}
		return nil
	}

	if err := walker.Walk(paths, preVisit, nil); err != nil {
		return fmt.Errorf("error walking files: %v", err)
	}

	return nil
}

func (s *Stats) PrintStats(config Config, processor middlewares.Processor) error {
	ctx := context.Background()

	if processor != nil {
		if config.OutputFlags&OutputOverview != 0 {
			if err := s.printOverviewAndFileTypes(ctx, processor); err != nil {
				return err
			}
		}
		if config.OutputFlags&OutputDirStructure != 0 {
			if err := s.printDirStructure(ctx, processor); err != nil {
				return err
			}
		}
		if config.OutputFlags&OutputFullStructure != 0 {
			if err := s.printFullStructure(ctx, processor); err != nil {
				return err
			}
		}
		return nil
	}

	// Fallback to text-based output if no processor is provided
	if config.OutputFlags&OutputOverview != 0 {
		s.PrintOverview()
	}

	if config.OutputFlags&OutputFullStructure != 0 {
		s.PrintFullStructure()
	} else if config.OutputFlags&OutputDirStructure != 0 {
		s.PrintDirStructure()
	}

	return nil
}

func (s *Stats) printOverviewAndFileTypes(ctx context.Context, processor middlewares.Processor) error {
	// Add total stats
	if err := processor.AddRow(ctx, types.NewRow(
		types.MRP("Type", "Total"),
		types.MRP("Name", ""),
		types.MRP("FileCount", s.Total.FileCount),
		types.MRP("TokenCount", s.Total.TokenCount),
		types.MRP("LineCount", s.Total.LineCount),
		types.MRP("Size", s.Total.Size),
	)); err != nil {
		return fmt.Errorf("error adding total stats row: %v", err)
	}

	// Add file type stats
	for ext, typeStats := range s.FileTypes {
		if err := processor.AddRow(ctx, types.NewRow(
			types.MRP("Type", "FileType"),
			types.MRP("Name", ext),
			types.MRP("FileCount", typeStats.FileCount),
			types.MRP("TokenCount", typeStats.TokenCount),
			types.MRP("LineCount", typeStats.LineCount),
			types.MRP("Size", typeStats.Size),
		)); err != nil {
			return fmt.Errorf("error adding file type stats row: %v", err)
		}
	}

	return nil
}

func (s *Stats) printDirStructure(ctx context.Context, processor middlewares.Processor) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %v", err)
	}

	dirs := s.getSortedDirs()
	for _, dir := range dirs {
		dirStats := s.Dirs[dir]
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil {
			relDir = dir // Fallback to absolute path if relative path can't be determined
		}
		if err := processor.AddRow(ctx, types.NewRow(
			types.MRP("Type", "Directory"),
			types.MRP("Name", relDir),
			types.MRP("FileCount", dirStats.FileCount),
			types.MRP("TokenCount", dirStats.TokenCount),
			types.MRP("LineCount", dirStats.LineCount),
			types.MRP("Size", dirStats.Size),
		)); err != nil {
			return fmt.Errorf("error adding directory stats row: %v", err)
		}
	}

	return nil
}

func (s *Stats) printFullStructure(ctx context.Context, processor middlewares.Processor) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %v", err)
	}

	dirs := s.getSortedDirs()
	for _, dir := range dirs {
		dirStats := s.Dirs[dir]
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil {
			relDir = dir // Fallback to absolute path if relative path can't be determined
		}
		if err := processor.AddRow(ctx, types.NewRow(
			types.MRP("Type", "Directory"),
			types.MRP("Name", relDir),
			types.MRP("FileCount", dirStats.FileCount),
			types.MRP("TokenCount", dirStats.TokenCount),
			types.MRP("LineCount", dirStats.LineCount),
			types.MRP("Size", dirStats.Size),
		)); err != nil {
			return fmt.Errorf("error adding directory stats row: %v", err)
		}

		files := s.DirFiles[dir]
		sort.Strings(files)

		for _, file := range files {
			fileStats := s.Files[file]
			relFile, err := filepath.Rel(cwd, file)
			if err != nil {
				relFile = file // Fallback to absolute path if relative path can't be determined
			}
			if err := processor.AddRow(ctx, types.NewRow(
				types.MRP("Type", "File"),
				types.MRP("Name", relFile),
				types.MRP("TokenCount", fileStats.TokenCount),
				types.MRP("LineCount", fileStats.LineCount),
				types.MRP("Size", fileStats.Size),
			)); err != nil {
				return fmt.Errorf("error adding file stats row: %v", err)
			}
		}
	}

	return nil
}

func (s *Stats) PrintOverview() {
	fmt.Println("Overview:")
	fmt.Printf("Total Files: %d\n", s.Total.FileCount)
	fmt.Printf("Total Directories: %d\n", len(s.Dirs))
	fmt.Printf("Total Tokens: %d\n", s.Total.TokenCount)
	fmt.Printf("Total Lines: %d\n", s.Total.LineCount)
	fmt.Printf("Total Size: %d bytes\n", s.Total.Size)

	fmt.Println("\nFile Type Statistics:")
	for ext, typeStats := range s.FileTypes {
		fmt.Printf("  %s:\n    Files: %d, Tokens: %d, Lines: %d, Size: %d bytes\n",
			ext, typeStats.FileCount, typeStats.TokenCount, typeStats.LineCount, typeStats.Size)
	}
	fmt.Println()
}

func (s *Stats) PrintDirStructure() {
	fmt.Println("Directory Structure:")
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		cwd = "" // Set to empty string to fall back to absolute paths
	}

	dirs := s.getSortedDirs()
	for _, dir := range dirs {
		dirStats := s.Dirs[dir]
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil {
			relDir = dir // Fallback to absolute path if relative path can't be determined
		}
		fmt.Printf("%s:\n  Files: %d, Tokens: %d, Lines: %d, Size: %d bytes\n",
			relDir, dirStats.FileCount, dirStats.TokenCount, dirStats.LineCount, dirStats.Size)
	}
	fmt.Println()
}

func (s *Stats) PrintFullStructure() {
	fmt.Println("Full Structure:")
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		cwd = "" // Set to empty string to fall back to absolute paths
	}

	dirs := s.getSortedDirs()
	for _, dir := range dirs {
		dirStats := s.Dirs[dir]
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil {
			relDir = dir // Fallback to absolute path if relative path can't be determined
		}
		fmt.Printf("%s:\n  Tokens: %d, Lines: %d, Size: %d bytes\n",
			relDir, dirStats.TokenCount, dirStats.LineCount, dirStats.Size)

		files := s.DirFiles[dir]
		sort.Strings(files)

		for _, file := range files {
			fileStats := s.Files[file]
			relFile, err := filepath.Rel(cwd, file)
			if err != nil {
				relFile = file // Fallback to absolute path if relative path can't be determined
			}
			fmt.Printf("  %s:\n    Tokens: %d, Lines: %d, Size: %d bytes\n",
				relFile, fileStats.TokenCount, fileStats.LineCount, fileStats.Size)
		}
		fmt.Println()
	}
}

func (s *Stats) getSortedDirs() []string {
	dirs := make([]string, 0, len(s.Dirs))
	for dir := range s.Dirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

func (s *Stats) GetStats(path string) (FileStats, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert path to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get the absolute path, fall back to the original path
		absPath = path
	}

	// Check if the path is a file
	if stats, ok := s.Files[absPath]; ok {
		return stats, true
	}

	// Check if the path is a directory
	if stats, ok := s.Dirs[absPath]; ok {
		return stats, true
	}

	// Path not found
	return FileStats{}, false
}
