package simulator

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DamageReport struct {
	Deleted   int
	Corrupted int
	Files     []string
}

// CorruptChunks overwrites random bytes in random files within the objects directory
func CorruptChunks(repoDir string, rate float64) (*DamageReport, error) {
	report := &DamageReport{}
	objectsDir := filepath.Join(repoDir, "objects")

	err := filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Skip hidden system files if any, but objects are usually hex
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Roll dice
		if shouldAct(rate) {
			if err := corruptFile(path); err != nil {
				return err
			}
			report.Corrupted++
			report.Files = append(report.Files, fmt.Sprintf("CORRUPTED: %s", filepath.Base(path)))
		}
		return nil
	})

	return report, err
}

// DeleteChunks deletes random files within the objects directory
func DeleteChunks(repoDir string, rate float64) (*DamageReport, error) {
	report := &DamageReport{}
	objectsDir := filepath.Join(repoDir, "objects")

	err := filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if shouldAct(rate) {
			if err := os.Remove(path); err != nil {
				return err
			}
			report.Deleted++
			report.Files = append(report.Files, fmt.Sprintf("DELETED: %s", filepath.Base(path)))
		}
		return nil
	})

	return report, err
}

func shouldAct(rate float64) bool {
	// Simple random float 0.0-1.0
	// crypto/rand is overkill for simulation but let's use math/rand logic or just bite byte
	b := make([]byte, 1)
	rand.Read(b)
	// byte is 0-255.
	val := float64(b[0]) / 255.0
	return val < rate
}

func corruptFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Corrupt first 100 bytes or random
	b := make([]byte, 50)
	rand.Read(b)

	// Seek to random position? Or just start. Start is fine, header corruption is worst case.
	_, err = f.WriteAt(b, 0)
	return err
}
