package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pranavdwivedi/aegis/pkg/chunker"
	"github.com/pranavdwivedi/aegis/pkg/crypto"
	"github.com/pranavdwivedi/aegis/pkg/index"
	"github.com/pranavdwivedi/aegis/pkg/intelligence"
	"github.com/pranavdwivedi/aegis/pkg/security"
	"github.com/pranavdwivedi/aegis/pkg/storage"
)

// Backup performs a backup of the sourcePath.
// repoDir is used for the Index (always local). backend is used for the chunks.
func Backup(repoDir string, backend storage.Backend, key crypto.MasterKey, sourcePath string) (int64, error) {
	security.RepoDir = repoDir // Ensure set if called via lib
	security.LogAction("BACKUP_START", fmt.Sprintf("Backing up %s", sourcePath))

	// 1. Open Index (Local)
	idx, err := index.NewIndex(repoDir, key)
	if err != nil {
		return 0, fmt.Errorf("failed to open index: %w", err)
	}
	defer idx.Close()

	// 2. Open Store (Backend)
	store, err := storage.NewContentAddressableStore(backend, key)
	if err != nil {
		return 0, fmt.Errorf("failed to open store: %w", err)
	}

	// 2. Create Snapshot
	absPath, _ := filepath.Abs(sourcePath)
	snapshotID, err := idx.CreateSnapshot(fmt.Sprintf("Backup of %s", absPath))
	if err != nil {
		return 0, err
	}

	// 3. Walk Files
	info, err := os.Stat(sourcePath)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		if err := processFile(sourcePath, snapshotID, idx, store); err != nil {
			return 0, err
		}
	} else {
		err = filepath.Walk(sourcePath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if err := processFile(p, snapshotID, idx, store); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return snapshotID, nil
}

func processFile(path string, snapshotID int64, idx *index.Index, store *storage.ContentAddressableStore) error {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Skipping %s: %v\n", path, err)
		return nil
	}
	defer f.Close()

	info, _ := f.Stat()
	fileID, err := idx.AddFile(snapshotID, path, info.Size(), uint32(info.Mode()), info.ModTime())
	if err != nil {
		return err
	}

	// Risk Analysis
	risk := intelligence.AnalyzeFile(path)
	if risk.Level == intelligence.RiskCritical || risk.Level == intelligence.RiskHigh {
		fmt.Printf("  [!] %s file detected: %s\n", risk.Level, filepath.Base(path))
	}

	// Chunking
	chnk := chunker.NewFixedSizeChunker(f, chunker.DefaultChunkSize)
	var offset int64 = 0

	for {
		chunk, err := chnk.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			break
		}

		// Store Chunk
		_, err = store.Put(chunk.Data)
		if err != nil {
			return err
		}

		// Index Chunk
		err = idx.AddChunk(fileID, chunk.Hash, offset, int64(len(chunk.Data)))
		if err != nil {
			return err
		}

		offset += int64(len(chunk.Data))
		// Optional: progress bar callback? For now silent logic or minimal output
		// fmt.Printf("\rProcessed: %s (%d bytes)", filepath.Base(path), offset)
	}
	fmt.Printf("Processed: %s\n", filepath.Base(path))
	return nil
}
