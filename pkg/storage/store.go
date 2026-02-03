package storage

import (
	"fmt"

	"github.com/klauspost/compress/zstd"
	"github.com/pranavdwivedi/aegis/pkg/crypto"
	"github.com/pranavdwivedi/aegis/pkg/hash"
)

// ContentAddressableStore implements a simple CAS on disk with Zstd compression
type ContentAddressableStore struct {
	backend Backend
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	key     crypto.MasterKey
}

// NewContentAddressableStore creates a new CAS
func NewContentAddressableStore(backend Backend, key crypto.MasterKey) (*ContentAddressableStore, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	return &ContentAddressableStore{
		backend: backend,
		encoder: enc,
		decoder: dec,
		key:     key,
	}, nil
}

// Put checks if object exists, if not, compresses, ENCRYPTS and writes it
func (s *ContentAddressableStore) Put(data []byte) (hash.Hash, error) {
	h := hash.Sum(data)
	keyStr := h.String()

	// Check exist
	if exists, _ := s.backend.Has(keyStr); exists {
		return h, nil
	}

	// 1. Compress
	compressed := s.encoder.EncodeAll(data, make([]byte, 0, len(data)))

	// 2. Encrypt
	encrypted, err := s.key.Encrypt(compressed)
	if err != nil {
		return hash.Hash{}, fmt.Errorf("encryption failed: %w", err)
	}

	// Write
	if err := s.backend.Put(keyStr, encrypted); err != nil {
		return hash.Hash{}, err
	}

	return h, nil
}

// Get retrieves, decrypts, and decompresses data
func (s *ContentAddressableStore) Get(h hash.Hash) ([]byte, error) {
	keyStr := h.String()

	// Read
	encrypted, err := s.backend.Get(keyStr)
	if err != nil {
		return nil, err
	}

	// 1. Decrypt
	compressed, err := s.key.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or data corruption): %w", err)
	}

	// 2. Decompress
	data, err := s.decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, err
	}

	// Verify integrity
	if hash.Sum(data) != h {
		return nil, fmt.Errorf("integrity check failed for chunk %s", h)
	}

	return data, nil
}

// Has checks if the chunk exists
func (s *ContentAddressableStore) Has(h hash.Hash) (bool, error) {
	return s.backend.Has(h.String())
}

// Verify checks the integrity of a stored chunk on disk
func (s *ContentAddressableStore) Verify(h hash.Hash) error {
	_, err := s.Get(h)
	return err
}
