package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pranavdwivedi/aegis/pkg/storage"
)

// Sync synchronizes all objects from source to dest.
// It assumes source is LocalBackend (to list files) and dest is generic Backend.
// For S3-to-Local sync, we would need a layout-aware Lister in the interface,
// but for now we focus on Backup Sync (Local -> Cloud).
func Sync(localRepoDir string, dest storage.Backend) error {
	objectsDir := filepath.Join(localRepoDir, "objects")

	type task struct {
		key  string
		path string
	}
	tasks := make(chan task, 100)
	var wg sync.WaitGroup

	// Worker pool
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				// Check if exists in dest
				exists, err := dest.Has(t.key)
				if err != nil {
					fmt.Printf("Error checking %s: %v\n", t.key, err)
					continue
				}
				if exists {
					// fmt.Printf("Skipping %s (exists)\n", t.key) // Verbose
					continue
				}

				// Upload
				data, err := os.ReadFile(t.path)
				if err != nil {
					fmt.Printf("Error reading %s: %v\n", t.path, err)
					continue
				}
				if err := dest.Put(t.key, data); err != nil {
					fmt.Printf("Error uploading %s: %v\n", t.key, err)
					continue
				}
				fmt.Printf("Synced: %s\n", t.key)
			}
		}()
	}

	start := time.Now()
	count := 0

	err := filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Calculate key from path logic?
		// LocalBackend stores as objects/ab/cdef...
		// Key is abcdef...
		// We need to reconstruct the key.
		// Rel path: ab/cdef...
		rel, _ := filepath.Rel(objectsDir, path)
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) != 2 {
			// Unexpected structure
			return nil
		}
		key := parts[0] + parts[1]

		tasks <- task{key: key, path: path}
		count++
		return nil
	})

	close(tasks)
	wg.Wait()

	fmt.Printf("Sync complete. Processed %d objects in %s.\n", count, time.Since(start))
	return err
}
