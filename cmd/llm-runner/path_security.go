package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafePath = errors.New("unsafe path")

type rootedReadCloser struct {
	file *os.File
	root *os.Root
}

func (r *rootedReadCloser) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *rootedReadCloser) Close() error {
	fileErr := r.file.Close()
	rootErr := r.root.Close()
	if fileErr != nil {
		return fileErr
	}
	return rootErr
}

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

// openFileUnderBase opens a file within baseDir using os.Root confinement.
func openFileUnderBase(baseDir, relPath string) (io.ReadCloser, string, error) {
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
	file, err := root.Open(relToBase)
	if err != nil {
		_ = root.Close()
		return nil, "", err
	}

	return &rootedReadCloser{file: file, root: root}, resolvedPath, nil
}

// readFileUnderBase reads a file within baseDir using os.Root confinement.
func readFileUnderBase(baseDir, relPath string) ([]byte, string, error) {
	reader, resolvedPath, err := openFileUnderBase(baseDir, relPath)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = reader.Close()
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}

	return data, resolvedPath, nil
}
