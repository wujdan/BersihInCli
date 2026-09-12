package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"storage-optimizer/internal/models"
)

func sampleReport() *models.ScanReport {
	return &models.ScanReport{
		Candidates: []*models.Candidate{
			{Meta: &models.FileMeta{Path: "D:\\Data\\logs\\b.log", Size: 500}, Category: models.CategoryLog, Label: models.LabelSafe, Confidence: 0.94, Reason: "log lama"},
			{Meta: &models.FileMeta{Path: "D:\\Data\\a.tmp", Size: 100}, Category: models.CategoryCache, Label: models.LabelSafe, Confidence: 0.96, Reason: "cache"},
			{Meta: &models.FileMeta{Path: "D:\\Data\\dup.bin", Size: 1024}, Category: models.CategoryDuplicate, Label: models.LabelReview, Confidence: 0.88, DuplicateGroup: "dup-abc", Reason: "duplikat"},
		},
	}
}

func TestExportReport_CSV(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.csv")
	if err := exportReport(sampleReport(), p); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{"path,category,label", "D:\\Data\\a.tmp", "D:\\Data\\dup.bin", "0.960"} {
		if !contains(s, want) {
			t.Errorf("CSV missing %q:\n%s", want, s)
		}
	}
}

func TestExportReport_JSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.json")
	if err := exportReport(sampleReport(), p); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var rows []exportRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[0].Path != "D:\\Data\\a.tmp" {
		t.Errorf("first row should be sorted by path, got %q", rows[0].Path)
	}
}

func TestExportReport_UnsupportedExt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.txt")
	if err := exportReport(sampleReport(), p); err == nil {
		t.Error("want error for unsupported extension")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}