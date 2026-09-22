package quarantine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"storage-optimizer/internal/config"
	"storage-optimizer/internal/models"
)

func testCfg(dir string) *config.Config {
	cfg := config.Default()
	cfg.App.DataDir = dir
	return cfg
}

func TestMoveFileMovesAndRecordsManifest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "f.tmp")
	content := "hello quarantine"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(src)
	cfg := testCfg(dir)
	q, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
	c := &models.Candidate{Meta: meta, Category: models.CategoryCache, Label: models.LabelSafe}
	m, err := q.MoveFile(c)
	if err != nil {
		t.Fatal("MoveFile error:", err)
	}
	// original file must be gone
	if _, err := os.Stat(src); err == nil {
		t.Error("source file should no longer exist after move")
	}
	// quarantine file must exist
	if _, err := os.Stat(m.QuarantinePath); err != nil {
		t.Error("quarantine file should exist:", err)
	}
	// SHA-256 recorded matches the content
	expected := sha256.Sum256([]byte(content))
	if m.SHA256 != hex.EncodeToString(expected[:]) {
		t.Errorf("SHA-256 mismatch: %s vs %s", m.SHA256, hex.EncodeToString(expected[:]))
	}
	if m.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", m.Size, len(content))
	}
	if m.OriginalPath != src {
		t.Errorf("OriginalPath = %q, want %q", m.OriginalPath, src)
	}
	// manifest persisted on disk
	if q.Count() != 1 {
		t.Errorf("Count = %d, want 1", q.Count())
	}
}

func TestRestoreMovesBack(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "restore_me.tmp")
	content := "restore this"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(src)
	cfg := testCfg(dir)
	q, _ := New(cfg)
	meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
	c := &models.Candidate{Meta: meta, Category: models.CategoryLog, Label: models.LabelSafe}
	m, err := q.MoveFile(c)
	if err != nil {
		t.Fatal(err)
	}

	// restore
	rm, err := q.Restore(m.ID)
	if err != nil {
		t.Fatal("Restore error:", err)
	}
	// quarantine file gone
	if _, err := os.Stat(rm.QuarantinePath); err == nil {
		t.Error("quarantine file should be gone after restore")
	}
	// original file restored
	got, err := os.ReadFile(rm.OriginalPath)
	if err != nil {
		t.Fatal("restored file missing:", err)
	}
	if string(got) != content {
		t.Errorf("restored content = %q, want %q", string(got), content)
	}
	if q.Count() != 0 {
		t.Errorf("Count = %d, want 0", q.Count())
	}
}

func TestPurgeExpired_NotExpiredStays(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "keep.tmp")
	os.WriteFile(src, []byte("data"), 0o644)
	info, _ := os.Stat(src)
	cfg := testCfg(dir) // default retention: cache = 14 days
	q, _ := New(cfg)
	meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
	c := &models.Candidate{Meta: meta, Category: models.CategoryCache, Label: models.LabelSafe}
	m, _ := q.MoveFile(c)

	// file still within retention must survive a plain `purge`.
	count, _, err := q.PurgeExpired(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("in-retention file should not be purged without --now, got count=%d", count)
	}
	if _, err := os.Stat(m.QuarantinePath); err != nil {
		t.Error("file should still exist (not expired yet)")
	}
}

func TestPurgeExpired_ForcePurgesAll(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "force.tmp")
	os.WriteFile(src, []byte("data"), 0o644)
	info, _ := os.Stat(src)
	cfg := testCfg(dir)
	q, _ := New(cfg)
	meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
	c := &models.Candidate{Meta: meta, Category: models.CategoryCache, Label: models.LabelSafe}
	m, _ := q.MoveFile(c)

	count, freed, err := q.PurgeExpired(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("want count=1, got %d", count)
	}
	if freed != m.Size {
		t.Errorf("freed=%d, want %d", freed, m.Size)
	}
	if _, err := os.Stat(m.QuarantinePath); err == nil {
		t.Error("file should be gone after purge")
	}
	if q.Count() != 0 {
		t.Errorf("Count = %d, want 0", q.Count())
	}
}

func TestPurgeExpired_SingleID(t *testing.T) {
	dir := t.TempDir()
	src1 := filepath.Join(dir, "a.tmp")
	src2 := filepath.Join(dir, "b.tmp")
	os.WriteFile(src1, []byte("1111"), 0o644)
	os.WriteFile(src2, []byte("2222"), 0o644)
	info1, _ := os.Stat(src1)
	info2, _ := os.Stat(src2)
	cfg := testCfg(dir)
	q, _ := New(cfg)

	move := func(src string, info os.FileInfo, cat models.Category) *Manifest {
		meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
		c := &models.Candidate{Meta: meta, Category: cat, Label: models.LabelSafe}
		m, err := q.MoveFile(c)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	m1 := move(src1, info1, models.CategoryCache)
	move(src2, info2, models.CategoryCache)

	count, _, err := q.PurgeExpired([]string{m1.ID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("want count=1, got %d", count)
	}
	if q.Count() != 1 {
		t.Errorf("want 1 remaining, got %d", q.Count())
	}
}

func TestPersistWritesManifestToDisk(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	q, _ := New(cfg)

	src := filepath.Join(dir, "p.tmp")
	os.WriteFile(src, []byte("batched"), 0o644)
	info, _ := os.Stat(src)
	meta := &models.FileMeta{Path: src, Size: info.Size(), ModTime: info.ModTime(), Extension: ".tmp"}
	c := &models.Candidate{Meta: meta, Category: models.CategoryCache, Label: models.LabelSafe}
	if _, err := q.MoveFile(c); err != nil {
		t.Fatal(err)
	}
	// manifest belum ditulis otomatis per-file; Persist menulisnya sekaligus
	if err := q.Persist(); err != nil {
		t.Fatal("Persist error:", err)
	}
	raw, err := os.ReadFile(q.manifestsFile())
	if err != nil {
		t.Fatal("manifest file tidak ada setelah Persist:", err)
	}
	if len(raw) == 0 {
		t.Fatal("manifest kosong")
	}

	// reload dari disk harus memulihkan isi yang sama
	q2, _ := New(cfg)
	if q2.Count() != 1 {
		t.Errorf("Count setelah reload = %d, want 1", q2.Count())
	}
}

func TestRetentionFor_PerCategory(t *testing.T) {
	cfg := config.Default()
	// cfg default: CacheTemp=14, LogDup=30, LargeOld=90
	q := &Dir{cfg: cfg}
	cases := []struct {
		cat models.Category
		want int
	}{
		{models.CategoryCache, 14},
		{models.CategoryLog, 30},
		{models.CategoryDuplicate, 30},
		{models.CategoryLargeOld, 90},
		{models.CategoryInstaller, 90},
		{models.CategoryOrphan, 90},
	}
	for _, tc := range cases {
		got := q.retentionFor(tc.cat)
		if got != tc.want {
			t.Errorf("retentionFor(%q) = %d, want %d", tc.cat, got, tc.want)
		}
	}
}

func TestListSortsByMovedAt(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	q, _ := New(cfg)
	now := time.Now()
	fixtures := []struct {
		id  string
		ts  time.Time
	}{
		{"1", now.Add(-2 * time.Hour)},
		{"2", now},
		{"3", now.Add(-1 * time.Hour)},
	}
	for _, f := range fixtures {
		q.manifests[f.id] = &Manifest{ID: f.id, MovedAt: f.ts}
	}
	list := q.List()
	if len(list) != 3 {
		t.Fatalf("want 3, got %d", len(list))
	}
	if list[0].ID != "2" || list[1].ID != "3" || list[2].ID != "1" {
		t.Errorf("unexpected order: %+v", list)
	}
}
