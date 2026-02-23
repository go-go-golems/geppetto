package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveSeedText(seed, seedFile string) (string, error) {
	if strings.TrimSpace(seed) != "" {
		return seed, nil
	}
	if strings.TrimSpace(seedFile) == "" {
		return "", nil
	}
	blob, err := os.ReadFile(seedFile)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

func loadDataset(path string) ([]any, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("dataset path is empty")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jsonl":
		return loadJSONL(path)
	case ".json":
		return loadJSONArray(path)
	default:
		// Try jsonl first, then json.
		if xs, err := loadJSONL(path); err == nil && len(xs) > 0 {
			return xs, nil
		}
		return loadJSONArray(path)
	}
}

func loadJSONL(path string) ([]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	closeWithErr := func(retErr error) error {
		cerr := f.Close()
		if retErr != nil {
			return retErr
		}
		return cerr
	}

	var out []any
	sc := bufio.NewScanner(f)
	// Allow fairly long lines.
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 10*1024*1024)

	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, closeWithErr(fmt.Errorf("jsonl parse error at line %d: %w", lineNo, err))
		}
		out = append(out, v)
	}
	if err := sc.Err(); err != nil {
		return nil, closeWithErr(err)
	}
	if err := closeWithErr(nil); err != nil {
		return nil, err
	}
	return out, nil
}

func loadJSONArray(path string) ([]any, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(blob, &v); err != nil {
		return nil, err
	}
	arr, ok := v.([]any)
	if ok {
		return arr, nil
	}
	if arr2, ok := v.([]interface{}); ok {
		out := make([]any, 0, len(arr2))
		out = append(out, arr2...)
		return out, nil
	}
	return nil, fmt.Errorf("json dataset must be an array, got %T", v)
}
