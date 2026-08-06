// Package config embeds and installs the sample epp-id configuration tree.
package config

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	embeddedRoot    = ".epp"
	homePlaceholder = "/home/CHANGE_ME_USERNAME"
)

//go:embed all:.epp
var sampleFiles embed.FS

// Install writes missing sample files to .epp under homeDir. Existing files
// are never overwritten. It returns the embedded paths that were created.
func Install(homeDir string) ([]string, error) {
	if homeDir == "" {
		return nil, fmt.Errorf("home directory is empty")
	}

	created := make([]string, 0)
	err := fs.WalkDir(sampleFiles, embeddedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		destination := filepath.Join(homeDir, filepath.FromSlash(path))
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0700); err != nil {
				return fmt.Errorf("create configuration directory %s: %w", destination, err)
			}
			return nil
		}

		data, err := sampleFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded configuration %s: %w", path, err)
		}
		if path == filepath.ToSlash(filepath.Join(embeddedRoot, "credentials")) {
			data = bytes.ReplaceAll(data, []byte(homePlaceholder), []byte(homeDir))
		}

		mode := fs.FileMode(0644)
		if filepath.Base(path) == "credentials" || filepath.Ext(path) == ".key" {
			mode = 0600
		}

		file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if os.IsExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("create configuration file %s: %w", destination, err)
		}

		written, writeErr := file.Write(data)
		if writeErr == nil && written != len(data) {
			writeErr = io.ErrShortWrite
		}
		if writeErr != nil {
			_ = file.Close()
			_ = os.Remove(destination)
			return fmt.Errorf("write configuration file %s: %w", destination, writeErr)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close configuration file %s: %w", destination, err)
		}

		created = append(created, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

// Created reports whether path was created by the most recent Install call.
func Created(paths []string, path string) bool {
	for _, createdPath := range paths {
		if createdPath == path {
			return true
		}
	}
	return false
}
