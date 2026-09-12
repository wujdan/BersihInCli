package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"storage-optimizer/internal/models"
)

// exportRow is a flat, serializable view of one candidate.
type exportRow struct {
	Path           string  `json:"path"`
	Category       string  `json:"category"`
	Label          string  `json:"label"`
	Confidence     float64 `json:"confidence"`
	SizeBytes      int64   `json:"size_bytes"`
	DuplicateGroup string  `json:"duplicate_group,omitempty"`
	Reason         string  `json:"reason"`
}

// exportReport writes all candidates to a CSV, TSV or JSON file (P3 feature).
// Format is inferred from the file extension.
func exportReport(r *models.ScanReport, path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".csv" && ext != ".tsv" && ext != ".json" {
		return fmt.Errorf("format file tidak didukung: %s (pakai .csv, .tsv, atau .json)", ext)
	}

	rows := make([]exportRow, 0, len(r.Candidates))
	for _, c := range r.Candidates {
		rows = append(rows, exportRow{
			Path:           c.Meta.Path,
			Category:       string(c.Category),
			Label:          string(c.Label),
			Confidence:     c.Confidence,
			SizeBytes:      c.Meta.Size,
			DuplicateGroup: c.DuplicateGroup,
			Reason:         c.Reason,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch ext {
	case ".json":
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rows); err != nil {
			return err
		}
	case ".csv":
		writeDelimited(f, rows, ',')
	case ".tsv":
		writeDelimited(f, rows, '\t')
	}
	return nil
}

func writeDelimited(f *os.File, rows []exportRow, sep rune) error {
	w := csv.NewWriter(f)
	w.Comma = sep
	if err := w.Write([]string{"path", "category", "label", "confidence", "size_bytes", "duplicate_group", "reason"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := w.Write([]string{
			r.Path, r.Category, r.Label,
			fmt.Sprintf("%.3f", r.Confidence),
			fmt.Sprintf("%d", r.SizeBytes),
			r.DuplicateGroup, r.Reason,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}