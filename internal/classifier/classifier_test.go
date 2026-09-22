package classifier

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"storage-optimizer/internal/config"
	"storage-optimizer/internal/models"
)

func testCfg() *config.Config {
	return config.Default()
}

// meta builds a FileMeta with a given age in days.
func meta(path string, size int64, ageDays int) *models.FileMeta {
	return &models.FileMeta{
		Path:      path,
		Size:      size,
		ModTime:   time.Now().AddDate(0, 0, -ageDays),
		Extension: strings.ToLower(filepath.Ext(path)),
	}
}

func TestClassify_ImportantExtensionsNeverCandidates(t *testing.T) {
	c := New(testCfg())
	important := []string{
		"D:\\code\\main.go",
		"D:\\docs\\laporan.pdf",
		"D:\\pics\\liburan.jpg",
		"D:\\video\\film.mp4",
		"D:\\music\\lagu.mp3",
		"D:\\proj\\data.db",
		"D:\\proj\\data.sqlite",
		"D:\\proj\\package.json",
		"D:\\proj\\notes.txt",
		"C:\\Users\\x\\.env",
		"C:\\Users\\x\\.gitignore",
	}
	for _, p := range important {
		// large + old on purpose: these must be protected regardless.
		if got := c.Classify(meta(p, 500*1024*1024, 400), ""); got != nil {
			t.Errorf("Classify(%s) = %+v, want nil (important)", p, got)
		}
	}
}

func TestClassify_DirNeverCandidate(t *testing.T) {
	c := New(testCfg())
	f := &models.FileMeta{Path: "C:\\temp\\folder", Size: 0, IsDir: true}
	if got := c.Classify(f, ""); got != nil {
		t.Errorf("Classify(dir) = %+v, want nil", got)
	}
}

func TestClassify_CacheByExtension(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("C:\\Users\\x\\AppData\\Local\\f.tmp", 100, 1), "")
	if got == nil {
		t.Fatal("want cache candidate, got nil")
	}
	if got.Category != models.CategoryCache {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryCache)
	}
	if got.Confidence != 0.96 {
		t.Errorf("confidence = %.2f, want 0.96", got.Confidence)
	}
	if got.Label != models.LabelSafe {
		t.Errorf("label = %q, want %q", got.Label, models.LabelSafe)
	}
}

func TestClassify_CacheByDirectory(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("C:\\Users\\x\\AppData\\Local\\CacheDir\\data.bin", 100, 1), "")
	if got == nil || got.Category != models.CategoryCache {
		t.Errorf("want cache candidate from cache dir, got %+v", got)
	}
}

func TestClassify_ThumbsDbIsCache(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("C:\\Users\\x\\Pictures\\thumbs.db", 50*1024, 1), "")
	if got == nil {
		t.Fatal("thumbs.db should be a cache candidate (regression)", )
	}
	if got.Category != models.CategoryCache {
		t.Errorf("thumbs.db category = %q, want %q", got.Category, models.CategoryCache)
	}
}

func TestClassify_OldLog(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("D:\\Data\\logs\\app.log", 500_000, 200), "")
	if got == nil {
		t.Fatal("want log candidate")
	}
	if got.Category != models.CategoryLog {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryLog)
	}
	if got.Confidence != 0.94 {
		t.Errorf("confidence = %.2f, want 0.94", got.Confidence)
	}
}

func TestClassify_RecentLogIsSafe(t *testing.T) {
	c := New(testCfg())
	if got := c.Classify(meta("D:\\Data\\logs\\app.log", 500_000, 10), ""); got != nil {
		t.Errorf("recent log should be safe, got %+v", got)
	}
}

func TestClassify_OldInstaller(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("D:\\Downloads\\setup.msi", 50*1024*1024, 200), "")
	if got == nil {
		t.Fatal("want installer candidate")
	}
	if got.Category != models.CategoryInstaller {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryInstaller)
	}
	if got.Confidence != 0.68 {
		t.Errorf("confidence = %.2f, want 0.68", got.Confidence)
	}
	if got.Label != models.LabelReview {
		t.Errorf("label = %q, want %q", got.Label, models.LabelReview)
	}
}

func TestClassify_RecentInstallerIsSafe(t *testing.T) {
	c := New(testCfg())
	if got := c.Classify(meta("D:\\Downloads\\setup.msi", 50*1024*1024, 10), ""); got != nil {
		t.Errorf("recent installer should be safe, got %+v", got)
	}
}

func TestClassify_Duplicate(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("D:\\Data\\dup1.bin", 5*1024*1024, 1), "dup-abcd1234")
	if got == nil {
		t.Fatal("want duplicate candidate")
	}
	if got.Category != models.CategoryDuplicate {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryDuplicate)
	}
	if got.Confidence != 0.88 {
		t.Errorf("confidence = %.2f, want 0.88", got.Confidence)
	}
	if got.DuplicateGroup != "dup-abcd1234" {
		t.Errorf("dup group = %q, want dup-abcd1234", got.DuplicateGroup)
	}
}

func TestClassify_RecentFileIsSafe(t *testing.T) {
	c := New(testCfg())
	if got := c.Classify(meta("D:\\Data\\f.bin", 10_000, 5), ""); got != nil {
		t.Errorf("recently modified file should be safe, got %+v", got)
	}
}

func TestClassify_LargeOldFile(t *testing.T) {
	c := New(testCfg())
	got := c.Classify(meta("D:\\Data\\video_archive.bin", 300*1024*1024, 400), "")
	if got == nil {
		t.Fatal("want large-old candidate")
	}
	if got.Category != models.CategoryLargeOld {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryLargeOld)
	}
	if got.Confidence != 0.55 {
		t.Errorf("confidence = %.2f, want 0.55", got.Confidence)
	}
}

func TestClassify_SmallOldFileIsSafe(t *testing.T) {
	c := New(testCfg())
	if got := c.Classify(meta("D:\\Data\\small_old.bin", 10_000, 400), ""); got != nil {
		t.Errorf("small old file should not be a candidate, got %+v", got)
	}
}

func TestClassify_LargeRecentIsSafe(t *testing.T) {
	c := New(testCfg())
	if got := c.Classify(meta("D:\\Data\\big_recent.bin", 2*1024*1024*1024, 5), ""); got != nil {
		t.Errorf("large recently used file should be safe, got %+v", got)
	}
}

func TestClassify_LargeOldButRecentlyAccessedIsSafe(t *testing.T) {
	c := New(testCfg())
	f := meta("D:\\Data\\video_archive.bin", 300*1024*1024, 500)
	f.AccessTime = time.Now().AddDate(0, 0, -2) // read 2 hari lalu
	if got := c.Classify(f, ""); got != nil {
		t.Errorf("file besar yang baru diakses harus aman, got %+v", got)
	}
}

func TestClassify_LargeOldAndNotAccessedIsCandidate(t *testing.T) {
	c := New(testCfg())
	f := meta("D:\\Data\\video_archive.bin", 300*1024*1024, 500)
	f.AccessTime = time.Now().AddDate(0, 0, -500) // tak diakses 500 hari
	got := c.Classify(f, "")
	if got == nil {
		t.Fatal("want large-old candidate when not accessed for long")
	}
	if got.Category != models.CategoryLargeOld {
		t.Errorf("catgegory = %q, want %q", got.Category, models.CategoryLargeOld)
	}
}

func TestClassify_OrphanAppData(t *testing.T) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		t.Skip("LOCALAPPDATA tidak tersedia")
	}
	c := New(testCfg())
	// token acak dijamin tidak pernah terdaftar sebagai aplikasi terpasang
	token := "zzorphan_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	f := meta(filepath.Join(root, token, "data.bin"), 1024, 200)
	got := c.Classify(f, "")
	if got == nil {
		t.Fatal("want orphan candidate, got nil")
	}
	if got.Category != models.CategoryOrphan {
		t.Errorf("category = %q, want %q", got.Category, models.CategoryOrphan)
	}
	if got.Confidence != 0.62 {
		t.Errorf("confidence = %.2f, want 0.62", got.Confidence)
	}
	if got.Label != models.LabelReview {
		t.Errorf("label = %q, want %q", got.Label, models.LabelReview)
	}
}

func TestClassify_KnownSafeAppDataDirIsSafe(t *testing.T) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		t.Skip("LOCALAPPDATA tidak tersedia")
	}
	c := New(testCfg())
	// "Microsoft" masuk knownSafeFolderNames → tidak pernah diklaim orphan
	f := meta(filepath.Join(root, "Microsoft", "Windows", "data.bin"), 1024, 200)
	if got := c.Classify(f, ""); got != nil {
		t.Errorf("folder app-data milik sistem harus aman, got %+v", got)
	}
}