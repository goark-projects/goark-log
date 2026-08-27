package goarklog

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

func compressFile(path string) (string, error) {
	return compressFileTo(path, path+".gz")
}

func compressFileTo(path string, compressedPath string) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("goark-log: open archive log file %q: %w", path, err)
	}
	sourceClosed := false
	defer func() {
		if !sourceClosed {
			_ = source.Close()
		}
	}()
	target, err := os.OpenFile(compressedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("goark-log: create gzip archive %q: %w", compressedPath, err)
	}
	removeCompressed := true
	defer func() {
		if removeCompressed {
			_ = os.Remove(compressedPath)
		}
	}()
	gzipWriter := gzip.NewWriter(target)
	if _, err := io.Copy(gzipWriter, source); err != nil {
		_ = gzipWriter.Close()
		_ = target.Close()
		return "", fmt.Errorf("goark-log: gzip archive %q: %w", path, err)
	}
	if err := gzipWriter.Close(); err != nil {
		_ = target.Close()
		return "", fmt.Errorf("goark-log: close gzip archive %q: %w", compressedPath, err)
	}
	if err := source.Close(); err != nil {
		_ = target.Close()
		return "", fmt.Errorf("goark-log: close archive log file %q: %w", path, err)
	}
	sourceClosed = true
	if err := target.Close(); err != nil {
		return "", fmt.Errorf("goark-log: close gzip file %q: %w", compressedPath, err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("goark-log: remove uncompressed archive %q: %w", path, err)
	}
	removeCompressed = false
	return compressedPath, nil
}
