package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

type Entry struct {
	Path string
}

func NewEntry(path string) (*Entry, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist")
	}

	return &Entry{Path: path}, nil
}
