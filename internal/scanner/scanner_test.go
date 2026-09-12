package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"storage-optimizer/internal/config"
	"storage-optimizer/internal/models"
)

func testScanner(t *testing.T, custom []string) (*Scanner, *config.Config) {
	t.Helper()
	cfg := config.Default()
	cfg.Scan.WorkerCount = 2
	// Clear platform lists for deterministic tests: t.TempDir() on Windows
	// lives under AppData\Local\Temp which default excludes would skip.
	cfg.Exclude.Windows = nil
	cfg.Exclude.Linux = nil
	cfg.Exclude.Darwin = nil
	cfg.Exclude.Custom = custom
	s := New(cfg, nil)
	return s, cfg
}

func TestMatchPath_Wildcard(t *testing.T) {
	pattern := `c:\users\*\appdata\local\temp`
	path := `c:\users\joko\appdata\local\temp`
	if !matchPath(path, pattern) {
		t.Errorf("matchPath(%q, %q) = false, want true", path, pattern)
	}
	no := `c:\users\joko\appdata\local\other`
	if matchPath(no, pattern) {
		t.Errorf("matchPath(%q, %q) = true, want false", no, pattern)
	}
}

func TestMatchPath_Exact(t *testing.T) {
	if !matchPath(`c:\windows`, `c:\windows`) {
		t.Error("exact match should be true")
	}
	if matchPath(`c:\windows2`, `c:\windows`) {
		t.Error("partial prefix should not match")
	}
}

func TestIsExcludedDir_ExactMatch(t *testing.T) {
	root := t.TempDir()
	s, _ := testScanner(t, []string{filepath.Join(root, "cache")})
	if !s.isExcludedDir(filepath.Join(root, "cache")) {
		t.Error("exact excluded dir should be excluded")
	}
}

func TestIsExcludedDir_ChildExcluded(t *testing.T) {
	root := t.TempDir()
	s, _ := testScanner(t, []string{filepath.Join(root, "cache")})
	child := filepath.Join(root, "cache", "deep", "deeper")
	if !s.isExcludedDir(child) {
		t.Error("children of excluded dir must be excluded")
	}
}

func TestIsExcludedDir_NotExcluded(t *testing.T) {
	root := t.TempDir()
	s, _ := testScanner(t, []string{filepath.Join(root, "cache")})
	if s.isExcludedDir(filepath.Join(root, "keepme")) {
		t.Error("unrelated dir must not be excluded")
	}
}

func TestIsExcludedDir_WildcardSegment(t *testing.T) {
	s, _ := testScanner(t, []string{`C:\Users\*\AppData\Local\Temp`})
	if !s.isExcludedDir(`C:\Users\budi\AppData\Local\Temp`) {
		t.Error("wildcard excluded dir should match")
	}
}

func TestHashFilePrefix_Deterministic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	content := []byte("the quick brown fox jumps over the lazy dog")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])

	got, err := hashFilePrefix(p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("hash = %s, want %s", got, want)
	}
}

func TestScan_WalksAndRespectsExcludes(t *testing.T) {
	root := t.TempDir()
	mk := func(p string) string {
		path := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mk("keep.txt")
	mk("sub\\inside.log")
	mk("cache\\hidden.tmp") // excluded via custom list
	excluded := filepath.Join(root, "cache")

	s, _ := testScanner(t, []string{excluded})
	s.roots = []string{root}

	ch := make(chan models.FileMeta, 16)
	var mu sync.Mutex
	var got []models.FileMeta
	done := make(chan struct{})
	go func() {
		defer close(done)
		for f := range ch {
			mu.Lock()
			got = append(got, f)
			mu.Unlock()
		}
	}()
	count, _, _, err := s.Scan(context.Background(), ch)
	close(ch)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("filesScanned = %d, want 2", count)
	}
	for _, f := range got {
		if strings.Contains(strings.ToLower(f.Path), strings.ToLower("cache")) {
			t.Errorf("excluded file leaked into results: %s", f.Path)
		}
	}
}