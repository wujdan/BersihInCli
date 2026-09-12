package safety

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"storage-optimizer/internal/models"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func statMeta(t *testing.T, path string) *models.FileMeta {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return &models.FileMeta{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
}

func TestValidate_ValidFilePasses(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "f.tmp")
	writeFile(t, p, "hello")
	res := New().Validate(statMeta(t, p))
	if !res.Passed {
		t.Errorf("want pass, got %+v", res)
	}
}

func TestValidate_MissingFileFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing.tmp")
	res := New().Validate(&models.FileMeta{Path: p, Size: 1, ModTime: time.Now()})
	if res.Passed {
		t.Error("missing file must fail validation")
	}
	if res.Reason == "" {
		t.Error("missing reason")
	}
}

func TestValidate_DirectoryFails(t *testing.T) {
	dir := t.TempDir()
	res := New().Validate(&models.FileMeta{Path: dir, Size: 0, ModTime: time.Now()})
	if res.Passed {
		t.Error("directory must fail validation")
	}
}

func TestValidate_SizeMismatchFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.tmp")
	writeFile(t, p, "hello")
	meta := statMeta(t, p)
	meta.Size = meta.Size + 10 // pretend scan saw a different size
	res := New().Validate(meta)
	if res.Passed {
		t.Error("size mismatch must fail validation")
	}
}

func TestValidate_ModtimeChangedFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.tmp")
	writeFile(t, p, "hello")
	meta := statMeta(t, p)
	meta.ModTime = meta.ModTime.Add(-time.Hour) // changed since scan
	res := New().Validate(meta)
	if res.Passed {
		t.Error("stale modtime must fail validation")
	}
}

func TestValidate_SymlinkFails(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("symlink creation may require privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.tmp")
	link := filepath.Join(dir, "link.tmp")
	writeFile(t, target, "data")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	res := New().Validate(statMeta(t, link))
	if res.Passed {
		t.Error("symlink must fail validation")
	}
}

func TestIsLockedError(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"The process cannot access the file because it is being used by another process", true},
		{"The requested operation cannot be performed because there is a sharing violation on this file", true},
		{"Resource busy: /some/file", true},
		{"another process is using this file", true},
		{"permission denied", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isLockedError(errors.New(c.in)); got != c.want {
			t.Errorf("isLockedError(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}