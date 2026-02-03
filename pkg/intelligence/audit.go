package intelligence

import (
	"fmt"

	"github.com/pranavdwivedi/aegis/pkg/hash"
	"github.com/pranavdwivedi/aegis/pkg/index"
	"github.com/pranavdwivedi/aegis/pkg/storage"
)

type AuditReport struct {
	TotalFiles    int
	TotalChunks   int
	MissingChunks int
	CorruptChunks int
	Healthy       bool
	Score         int // 0-100
}

// AuditRepository checks every chunk in the repository for integrity
func AuditRepository(idx *index.Index, store *storage.ContentAddressableStore) (AuditReport, error) {
	snapshots, err := idx.ListSnapshots()
	if err != nil {
		return AuditReport{}, err
	}

	report := AuditReport{Healthy: true, Score: 100}
	checkedChunks := make(map[string]bool)

	for _, s := range snapshots {
		files, err := idx.GetFiles(s.ID)
		if err != nil {
			return report, fmt.Errorf("audit failed fetching files for snapshot %d: %w", s.ID, err)
		}

		report.TotalFiles += len(files)

		for _, f := range files {
			chunks, err := idx.GetChunks(f.ID)
			if err != nil {
				return report, fmt.Errorf("audit failed fetching chunks for file %d: %w", f.ID, err)
			}

			// In a real audit we might scan thousands of chunks.
			// Optimization: unique chunks only.
			for _, c := range chunks {
				if checkedChunks[c.Hash] {
					continue
				}
				checkedChunks[c.Hash] = true
				report.TotalChunks++

				h, err := hash.Parse(c.Hash)
				if err != nil {
					report.CorruptChunks++
					continue
				}

				// Verify Existence & Integrity (Store.Get does verification)
				// We use Has first to check existence cheaply? No, we really want to read bytes to check bitrot.
				// Store.Get() decrypts and verifies hash.
				_, err = store.Get(h)
				if err != nil {
					// Distinguish missing vs corrupt?
					// Store.Get returns error for both.
					// We can check existence first.
					exists, _ := store.Has(h)
					if !exists {
						report.MissingChunks++
						fmt.Printf("MISSING CHUNK: %s (File: %s)\n", c.Hash, f.Path)
					} else {
						report.CorruptChunks++
						fmt.Printf("CORRUPT CHUNK: %s (File: %s) - %v\n", c.Hash, f.Path, err)
					}
				}
			}
		}
	}

	if report.MissingChunks > 0 || report.CorruptChunks > 0 {
		report.Healthy = false
		report.Score = 0 // Simply fail score for now if any corruption
	}

	return report, nil
}
