package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMutex sync.Mutex
	RepoDir  string // Must be set by main
)

type AuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Details   string    `json:"details"`
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
}

func logPath() string {
	return filepath.Join(RepoDir, "security.log")
}

func getLastHash() (string, error) {
	f, err := os.Open(logPath())
	if os.IsNotExist(err) {
		return "0000000000000000000000000000000000000000000000000000000000000000", nil // Genesis hash
	}
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read backwards or just read all? For valid chain, we usually append.
	// For simplicity in this tool, we read line by line to find the last one.
	// In production, we'd use a seek or keep state.
	var lastLine string
	dec := json.NewDecoder(f)
	for dec.More() {
		var entry AuditEntry
		if err := dec.Decode(&entry); err != nil {
			// If file is corrupted, return error
			return "", err
		}
		lastLine = entry.Hash
	}
	if lastLine == "" {
		return "0000000000000000000000000000000000000000000000000000000000000000", nil
	}
	return lastLine, nil
}

func LogAction(action, details string) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if RepoDir == "" {
		return fmt.Errorf("security repo dir not set")
	}

	prevHash, err := getLastHash()
	if err != nil {
		return fmt.Errorf("failed to get last hash: %v", err)
	}

	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Action:    action,
		User:      os.Getenv("USER"), // Simple user tracking
		Details:   details,
		PrevHash:  prevHash,
	}

	// Compute Hash
	// Hash = SHA256(PrevHash + Timestamp + Action + User + Details)
	h := sha256.New()
	h.Write([]byte(entry.PrevHash))
	h.Write([]byte(entry.Timestamp.Format(time.RFC3339Nano)))
	h.Write([]byte(entry.Action))
	h.Write([]byte(entry.User))
	h.Write([]byte(entry.Details))
	entry.Hash = hex.EncodeToString(h.Sum(nil))

	// Write to file
	f, err := os.OpenFile(logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	return encoder.Encode(entry)
}

// VerifyChain checks the integrity of the audit log
func VerifyChain() (bool, error) {
	logMutex.Lock()
	defer logMutex.Unlock()

	f, err := os.Open(logPath())
	if os.IsNotExist(err) {
		return true, nil // Empty is valid
	}
	if err != nil {
		return false, err
	}
	defer f.Close()

	var prevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	dec := json.NewDecoder(f)
	line := 0

	for {
		var entry AuditEntry
		if err := dec.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			return false, fmt.Errorf("corrupt log line %d: %v", line, err)
		}

		// Verify Chain Link
		if entry.PrevHash != prevHash {
			return false, fmt.Errorf("broken chain at line %d: prev_hash mismatch", line)
		}

		// Verify Integrity
		h := sha256.New()
		h.Write([]byte(entry.PrevHash))
		h.Write([]byte(entry.Timestamp.Format(time.RFC3339Nano)))
		h.Write([]byte(entry.Action))
		h.Write([]byte(entry.User))
		h.Write([]byte(entry.Details))
		expectedHash := hex.EncodeToString(h.Sum(nil))

		if entry.Hash != expectedHash {
			return false, fmt.Errorf("integrity failure at line %d: hash mismatch", line)
		}

		prevHash = entry.Hash
		line++
	}

	return true, nil
}
