package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafePath = errors.New("unsafe path")

// secureJoinUnderBase resolves relPath under baseDir and rejects traversal/absolute paths.
func secureJoinUnderBase(baseDir, relPath string) (string, error) {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}

	cleanRel := filepath.Clean(relPath)
	if cleanRel == "." {
		return baseAbs, nil
	}
	if filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("%w: absolute paths are not allowed", ErrUnsafePath)
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: parent traversal is not allowed", ErrUnsafePath)
	}

	candidateAbs, err := filepath.Abs(filepath.Join(baseAbs, cleanRel))
	if err != nil {
		return "", fmt.Errorf("resolve candidate path: %w", err)
	}
	relToBase, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path escapes base dir", ErrUnsafePath)
	}

	return candidateAbs, nil
}

// readFileUnderBase reads a file within baseDir using os.Root confinement.
func readFileUnderBase(baseDir, relPath string) ([]byte, string, error) {
	resolvedPath, err := secureJoinUnderBase(baseDir, relPath)
	if err != nil {
		return nil, "", err
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve base dir: %w", err)
	}
	relToBase, err := filepath.Rel(baseAbs, resolvedPath)
	if err != nil {
		return nil, "", fmt.Errorf("compute relative path: %w", err)
	}

	root, err := os.OpenRoot(baseAbs)
	if err != nil {
		return nil, "", fmt.Errorf("open base root: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile(relToBase)
	if err != nil {
		return nil, "", err
	}

	return data, resolvedPath, nil
}
