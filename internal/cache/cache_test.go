package cache

import (
	"path/filepath"
	"testing"
)

func TestResolveDirUsesXDGCacheHome(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache")

	dir, err := resolveDir()
	if err != nil {
		t.Fatalf("resolveDir() error = %v", err)
	}

	want := filepath.Join("/tmp/xdg-cache", appName)
	if dir != want {
		t.Fatalf("resolveDir() = %q, want %q", dir, want)
	}
}

func TestResolveDirFallsBackToHomeCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "/tmp/alex")

	dir, err := resolveDir()
	if err != nil {
		t.Fatalf("resolveDir() error = %v", err)
	}

	want := filepath.Join("/tmp/alex", ".cache", appName)
	if dir != want {
		t.Fatalf("resolveDir() = %q, want %q", dir, want)
	}
}

func TestResolveDirRejectsRelativeXDGCacheHome(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "relative/cache")

	_, err := resolveDir()
	if err == nil {
		t.Fatal("resolveDir() error = nil, want error")
	}
}
