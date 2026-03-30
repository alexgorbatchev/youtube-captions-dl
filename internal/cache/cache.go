package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appName         = "youtube-captions-dl"
	cacheFileSuffix = ".v8.txt"
)

type Store struct {
	dir string
}

func NewStore() (*Store, error) {
	dir, err := resolveDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache directory %q: %w", dir, err)
	}

	return &Store{dir: dir}, nil
}

func (s *Store) Load(videoID string) (string, bool, error) {
	data, err := os.ReadFile(s.path(videoID))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading cache file for %s: %w", videoID, err)
	}

	return string(data), true, nil
}

func (s *Store) Save(videoID string, text string) error {
	tmpFile, err := os.CreateTemp(s.dir, videoID+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp cache file for %s: %w", videoID, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.WriteString(text); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing temp cache file for %s: %w", videoID, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp cache file for %s: %w", videoID, err)
	}

	if err := os.Rename(tmpPath, s.path(videoID)); err != nil {
		return fmt.Errorf("moving cache file into place for %s: %w", videoID, err)
	}

	cleanup = false
	return nil
}

func resolveDir() (string, error) {
	root := os.Getenv("XDG_CACHE_HOME")
	if root != "" {
		if !filepath.IsAbs(root) {
			return "", fmt.Errorf("XDG_CACHE_HOME must be an absolute path, got %q", root)
		}

		return filepath.Join(root, appName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory for XDG cache fallback: %w", err)
	}

	return filepath.Join(homeDir, ".cache", appName), nil
}

func (s *Store) path(videoID string) string {
	return filepath.Join(s.dir, videoID+cacheFileSuffix)
}
