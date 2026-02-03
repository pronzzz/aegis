package index

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pranavdwivedi/aegis/pkg/crypto" // Added for crypto.MasterKey
	"github.com/pranavdwivedi/aegis/pkg/hash"
)

// Index manages the metadata in SQLite
type Index struct {
	db  *sql.DB
	key crypto.MasterKey // Added key field
}

// NewIndex creates a new index
func NewIndex(basePath string, key crypto.MasterKey) (*Index, error) { // Added key parameter
	dbPath := filepath.Join(basePath, "index.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	idx := &Index{db: db, key: key} // Initialized key
	if err := idx.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return idx, nil
}

func (i *Index) Close() error {
	return i.db.Close()
}

func (i *Index) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			description TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER,
			path TEXT NOT NULL,
			size INTEGER NOT NULL,
			mode INTEGER,
			mod_time DATETIME,
			FOREIGN KEY(snapshot_id) REFERENCES snapshots(id)
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER,
			hash TEXT NOT NULL,
			offset INTEGER NOT NULL,
			size INTEGER NOT NULL,
			FOREIGN KEY(file_id) REFERENCES files(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(hash)`,
	}

	for _, q := range queries {
		if _, err := i.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// CreateSnapshot starts a new snapshot
func (i *Index) CreateSnapshot(desc string) (int64, error) {
	res, err := i.db.Exec("INSERT INTO snapshots (timestamp, description) VALUES (?, ?)", time.Now(), desc)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddFile adds a file to a snapshot
func (i *Index) AddFile(snapshotID int64, path string, size int64, mode uint32, modTime time.Time) (int64, error) {
	// Encrypt path
	encryptedPath, err := i.key.Encrypt([]byte(path))
	if err != nil {
		return 0, err
	}

	// We store encrypted path as a hex string or base64 to be safe in TEXT field,
	// but raw bytes might be okay in SQLite blob if defining column as BLOB.
	// However, Schema says TEXT. Let's use hex for safety/easier debugging view.
	encodedPath := hex.EncodeToString(encryptedPath)

	res, err := i.db.Exec(
		"INSERT INTO files (snapshot_id, path, size, mode, mod_time) VALUES (?, ?, ?, ?, ?)",
		snapshotID, encodedPath, size, mode, modTime,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddChunk adds a chunk reference to a file
func (i *Index) AddChunk(fileID int64, h hash.Hash, offset int64, size int64) error {
	_, err := i.db.Exec(
		"INSERT INTO chunks (file_id, hash, offset, size) VALUES (?, ?, ?, ?)",
		fileID, h.String(), offset, size,
	)
	return err
}

type FileSnapshot struct {
	ID      int64
	Path    string
	Size    int64
	ModTime time.Time
}

// ListSnapshots returns all snapshots
func (i *Index) ListSnapshots() ([]struct {
	ID   int64
	Time time.Time
	Desc string
}, error) {
	rows, err := i.db.Query("SELECT id, timestamp, description FROM snapshots ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []struct {
		ID   int64
		Time time.Time
		Desc string
	}

	for rows.Next() {
		var s struct {
			ID   int64
			Time time.Time
			Desc string
		}
		if err := rows.Scan(&s.ID, &s.Time, &s.Desc); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, nil
}

// FileRecord represents a file inside a snapshot
type FileRecord struct {
	ID      int64
	Path    string
	Size    int64
	Mode    uint32
	ModTime time.Time
}

// GetFiles returns all files for a given snapshot
func (i *Index) GetFiles(snapshotID int64) ([]FileRecord, error) {
	rows, err := i.db.Query("SELECT id, path, size, mode, mod_time FROM files WHERE snapshot_id = ?", snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		var encodedPath string
		if err := rows.Scan(&f.ID, &encodedPath, &f.Size, &f.Mode, &f.ModTime); err != nil {
			return nil, err
		}

		// Decrypt Path
		encryptedPath, err := hex.DecodeString(encodedPath)
		if err != nil {
			return nil, fmt.Errorf("metadata corruption (hex decode): %w", err)
		}
		decryptedPath, err := i.key.Decrypt(encryptedPath)
		if err != nil {
			return nil, fmt.Errorf("metadata corruption (decrypt): %w", err)
		}
		f.Path = string(decryptedPath)

		files = append(files, f)
	}
	return files, nil
}

// ChunkRecord represents a chunk of a file
type ChunkRecord struct {
	Hash   string
	Offset int64
	Size   int64
}

// GetChunks returns all chunks for a file, ordered by offset
func (i *Index) GetChunks(fileID int64) ([]ChunkRecord, error) {
	rows, err := i.db.Query("SELECT hash, offset, size FROM chunks WHERE file_id = ? ORDER BY offset ASC", fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []ChunkRecord
	for rows.Next() {
		var c ChunkRecord
		if err := rows.Scan(&c.Hash, &c.Offset, &c.Size); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}
