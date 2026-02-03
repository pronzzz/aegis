package restore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pranavdwivedi/aegis/pkg/hash"
	"github.com/pranavdwivedi/aegis/pkg/index"
	"github.com/pranavdwivedi/aegis/pkg/storage"
)

// RestoreSnapshot restores all files from a snapshot to the target directory
func RestoreSnapshot(idx *index.Index, store *storage.ContentAddressableStore, snapshotID int64, targetDir string, force bool, dryRun bool, priorityPatterns []string) error {
	// 1. Fetch File List
	files, err := idx.GetFiles(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to fetch files for snapshot %d: %w", snapshotID, err)
	}

	// 2. Sort files by priority logic
	sort.Slice(files, func(i, j int) bool {
		pI := getPriorityScore(files[i].Path, priorityPatterns)
		pJ := getPriorityScore(files[j].Path, priorityPatterns)
		if pI != pJ {
			return pI < pJ
		}
		return files[i].Path < files[j].Path
	})

	fmt.Printf("Restoring %d files to %s...\n", len(files), targetDir)

	for _, f := range files {
		// Determine absolute destination path
		// Remove leading / or relative components from f.Path to be safe?
		// Assuming f.Path is the absolute path where it was backed up from.
		// We want to restore it relative to targetDir if targetDir is specified.
		// However, if f.Path is absolute (e.g. /Users/foo/bar), relying on targetDir behavior needs definition.
		// For this implementation: "Restore to <targetDir>/<original_absolute_path_without_root>"
		// Example: Backup /etc/hosts -> Restore to ./restored/etc/hosts

		relPath := f.Path
		if filepath.IsAbs(f.Path) {
			// strip volume name if needed, but for now just strip leading separator
			if len(f.Path) > 0 && f.Path[0] == filepath.Separator {
				relPath = f.Path[1:]
			}
		}

		destPath := filepath.Join(targetDir, relPath)

		// Check existence
		if _, err := os.Stat(destPath); err == nil {
			if !force {
				return fmt.Errorf("file already exists: %s (use --force to overwrite)", destPath)
			}
		}

		if err := restoreFile(idx, store, f, destPath, dryRun); err != nil {
			return fmt.Errorf("failed to restore %s: %w", f.Path, err)
		}
		fmt.Printf("Restored: %s\n", destPath)
	}

	return nil
}

func restoreFile(idx *index.Index, store *storage.ContentAddressableStore, f index.FileRecord, destPath string, dryRun bool) error {
	// Fetch chunks
	chunks, err := idx.GetChunks(f.ID)
	if err != nil {
		return err
	}

	var out *os.File
	if !dryRun {
		// Ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
			return err
		}

		// Create file
		out, err = os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()
	}

	// Reassemble
	for _, c := range chunks {
		h, err := hash.Parse(c.Hash)
		if err != nil {
			return err
		}

		data, err := store.Get(h)
		if err != nil {
			return fmt.Errorf("chunk missing or corrupted %s: %w", c.Hash, err)
		}

		if !dryRun {
			if _, err := out.Write(data); err != nil {
				return err
			}
		}
	}

	if !dryRun {
		// Set permissions
		if err := out.Chmod(os.FileMode(f.Mode)); err != nil {
			return err
		}
		if err := os.Chtimes(destPath, time.Now(), f.ModTime); err != nil {
			// ignore error
		}
	}

	return nil
}

func getPriorityScore(path string, patterns []string) int {
	// Lower score = Higher Priority (Sorted first)
	for i, p := range patterns {
		// Try matching on the full path or just the filename
		matched, _ := filepath.Match(p, filepath.Base(path))
		if matched {
			return i
		}
	}
	// Default priority (lowest)
	return len(patterns)
}
