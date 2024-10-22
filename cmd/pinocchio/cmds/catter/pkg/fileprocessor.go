package pkg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-go-golems/clay/pkg/filefilter"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/weaviate/tiktoken-go"
)

type FileProcessor struct {
	MaxTotalSize  int64
	TotalSize     int64
	TotalTokens   int
	FileCount     int
	TokenCounter  *tiktoken.Tiktoken
	TokenCounts   map[string]int
	ListOnly      bool
	DelimiterType string
	MaxLines      int
	MaxTokens     int
	Filter        *filefilter.FileFilter
	PrintFilters  bool
	Processor     middlewares.Processor
	Stats         *Stats
}

type FileProcessorOption func(*FileProcessor)

var (
	ErrMaxTokensExceeded    = errors.New("maximum total tokens limit reached")
	ErrMaxTotalSizeExceeded = errors.New("maximum total size limit reached")
)

func NewFileProcessor(options ...FileProcessorOption) *FileProcessor {
	tokenCounter, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing tiktoken: %v\n", err)
		os.Exit(1)
	}

	fp := &FileProcessor{
		TokenCounter: tokenCounter,
		TokenCounts:  make(map[string]int),
		MaxLines:     0,
		MaxTokens:    0,
		PrintFilters: false,
	}

	for _, option := range options {
		option(fp)
	}

	return fp
}

func WithMaxTotalSize(size int64) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.MaxTotalSize = size
	}
}

func WithListOnly(listOnly bool) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.ListOnly = listOnly
	}
}

func WithDelimiterType(delimiterType string) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.DelimiterType = delimiterType
	}
}

func WithMaxLines(maxLines int) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.MaxLines = maxLines
	}
}

func WithMaxTokens(maxTokens int) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.MaxTokens = maxTokens
	}
}

func WithFileFilter(filter *filefilter.FileFilter) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.Filter = filter
	}
}

func WithPrintFilters(printFilters bool) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.PrintFilters = printFilters
	}
}

func WithProcessor(processor middlewares.Processor) FileProcessorOption {
	return func(fp *FileProcessor) {
		fp.Processor = processor
	}
}

func (fp *FileProcessor) ProcessPaths(paths []string) error {
	if fp.PrintFilters {
		fp.printConfiguredFilters()
		return nil
	}

	fp.Stats = NewStats()
	err := fp.Stats.ComputeStats(paths, fp.Filter)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error computing stats: %v\n", err)
		return err
	}

	for _, path := range paths {
		err := fp.processPath(path)
		if err != nil {
			if errors.Is(err, ErrMaxTokensExceeded) {
				_, _ = fmt.Fprintf(os.Stderr, "Reached maximum total tokens limit of %d\n", fp.MaxTokens)
				return nil
			} else if errors.Is(err, ErrMaxTotalSizeExceeded) {
				_, _ = fmt.Fprintf(os.Stderr, "Reached maximum total size limit of %d bytes\n", fp.MaxTotalSize)
				return nil
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "Error processing path %s: %v\n", path, err)
				return err
			}
		}
	}

	return nil
}

func (fp *FileProcessor) processPath(path string) error {
	if fp.MaxTokens != 0 && fp.TotalTokens >= fp.MaxTokens {
		return ErrMaxTokensExceeded
	}
	if fp.MaxTotalSize != 0 && fp.TotalSize >= fp.MaxTotalSize {
		return ErrMaxTotalSizeExceeded
	}

	if fp.Filter == nil || fp.Filter.FilterPath(path) {
		if fileInfo, err := os.Stat(path); err == nil {
			if fileInfo.IsDir() {
				return fp.processDirectory(path)
			} else {
				return fp.printFileContent(path)
			}
		}
	}

	return nil
}

func (fp *FileProcessor) processDirectory(dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", dirPath, err)
	}

	dirTokens := 0
	for _, file := range files {
		fullPath := filepath.Join(dirPath, file.Name())
		err := fp.processPath(fullPath)
		if err != nil {
			return err // Propagate the error up
		}
		dirTokens += fp.TokenCounts[fullPath]
	}
	fp.TokenCounts[dirPath] = dirTokens
	return nil
}

func (fp *FileProcessor) printFileContent(filePath string) error {
	if fp.ListOnly {
		fmt.Println(filePath)
		return nil
	}

	fileStats, ok := fp.Stats.GetStats(filePath)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Error: Stats not found for file %s\n", filePath)
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filePath, err)
		return nil
	}

	limitedContent := fp.applyLimits(content)
	actualSize := int64(len(limitedContent))
	actualTokenCount := len(fp.TokenCounter.Encode(limitedContent, nil, nil))

	fileStats.Size = actualSize
	fileStats.TokenCount = actualTokenCount

	fp.TokenCounts[filePath] = fileStats.TokenCount
	fp.TotalTokens += fileStats.TokenCount

	if fp.MaxTotalSize != 0 && fp.TotalSize+actualSize > fp.MaxTotalSize {
		remainingSize := fp.MaxTotalSize - fp.TotalSize
		if remainingSize > 0 {
			limitedContent = limitedContent[:remainingSize]
			actualSize = remainingSize
		} else {
			return ErrMaxTotalSizeExceeded
		}
	}
	if fp.MaxTokens != 0 && fp.TotalTokens+actualTokenCount > fp.MaxTokens {
		return ErrMaxTokensExceeded
	}

	actualLineCount := strings.Count(limitedContent, "\n")

	if fp.Processor != nil {
		ctx := context.Background()
		err := fp.Processor.AddRow(ctx, types.NewRow(
			types.MRP("Path", filePath),
			types.MRP("FileSize", fileStats.Size),
			types.MRP("FileTokenCount", fileStats.TokenCount),
			types.MRP("FileLineCount", fileStats.LineCount),
			types.MRP("ActualSize", actualSize),
			types.MRP("ActualTokenCount", actualTokenCount),
			types.MRP("ActualLineCount", actualLineCount),
			types.MRP("Content", limitedContent),
		))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error adding row to processor: %v\n", err)
		}
	} else {
		switch fp.DelimiterType {
		case "xml":
			fmt.Printf("<file name=\"%s\">\n<content>\n%s\n</content>\n</file>\n", filePath, limitedContent)
		case "markdown":
			fmt.Printf("## File: %s\n\n```\n%s\n```\n\n", filePath, limitedContent)
		case "simple":
			fmt.Printf("===\n\nFile: %s\n\n---\n\n%s\n\n===\n\n", filePath, limitedContent)
		default:
			fmt.Printf("=== BEGIN: %s ===\n%s\n=== END: %s ===\n\n", filePath, limitedContent, filePath)
		}
	}

	fp.TotalSize += actualSize
	fp.FileCount++

	return nil
}

func (fp *FileProcessor) applyLimits(content []byte) string {
	if fp.MaxLines == 0 && fp.MaxTokens == 0 {
		return string(content)
	}

	var limitedContent bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineCount := 0
	tokenCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineTokens := fp.TokenCounter.Encode(line, nil, nil)

		if fp.MaxLines > 0 && lineCount >= fp.MaxLines {
			break
		}

		if fp.MaxTokens > 0 && tokenCount+len(lineTokens) > fp.MaxTokens {
			remainingTokens := fp.MaxTokens - tokenCount
			if remainingTokens > 0 {
				decodedLine := fp.TokenCounter.Decode(lineTokens[:remainingTokens])
				limitedContent.WriteString(decodedLine)
			}
			break
		}

		limitedContent.WriteString(line + "\n")
		lineCount++
		tokenCount += len(lineTokens)
	}

	return limitedContent.String()
}

func (fp *FileProcessor) printConfiguredFilters() {
	fmt.Println("Configured Filters:")
	fmt.Println("-------------------")

	if fp.Filter == nil {
		fmt.Println("No filters configured.")
		return
	}

	fmt.Printf("Max File Size: %d bytes\n", fp.Filter.MaxFileSize)
	fmt.Printf("Disable Default Filters: %v\n", fp.Filter.DisableDefaultFilters)
	fmt.Printf("Disable GitIgnore: %v\n", fp.Filter.DisableGitIgnore)
	fmt.Printf("Filter Binary Files: %v\n", fp.Filter.FilterBinaryFiles)
	fmt.Printf("Verbose: %v\n", fp.Filter.Verbose)

	printStringList("Include Extensions", fp.Filter.IncludeExts)
	printStringList("Exclude Extensions", fp.Filter.ExcludeExts)
	printStringList("Exclude Directories", fp.Filter.ExcludeDirs)

	printRegexpList("Match Filenames", fp.Filter.MatchFilenames)
	printRegexpList("Match Paths", fp.Filter.MatchPaths)
	printRegexpList("Exclude Match Filenames", fp.Filter.ExcludeMatchFilenames)
	printRegexpList("Exclude Match Paths", fp.Filter.ExcludeMatchPaths)

	fmt.Println("\nFile Processor Settings:")
	fmt.Printf("Max Total Size: %d bytes\n", fp.MaxTotalSize)
	fmt.Printf("Max Lines: %d\n", fp.MaxLines)
	fmt.Printf("Max Tokens: %d\n", fp.MaxTokens)
	fmt.Printf("List Only: %v\n", fp.ListOnly)
	fmt.Printf("Delimiter Type: %s\n", fp.DelimiterType)
}

func printStringList(name string, list []string) {
	if len(list) > 0 {
		fmt.Printf("%s: %s\n", name, strings.Join(list, ", "))
	}
}

func printRegexpList(name string, list []*regexp.Regexp) {
	if len(list) > 0 {
		patterns := make([]string, len(list))
		for i, re := range list {
			patterns[i] = re.String()
		}
		fmt.Printf("%s: %s\n", name, strings.Join(patterns, ", "))
	}
}
