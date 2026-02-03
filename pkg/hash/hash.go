package hash

import (
	"encoding/hex"
	"io"

	"github.com/zeebo/blake3"
)

// Hash represents a 32-byte BLAKE3 hash
type Hash [32]byte

// String returns the hexadecimal representation of the hash
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Bytes returns the byte slice of the hash
func (h Hash) Bytes() []byte {
	return h[:]
}

// New returns a new BLAKE3 hasher
func New() *blake3.Hasher {
	return blake3.New()
}

// Sum computes the BLAKE3 hash of the given data
func Sum(data []byte) Hash {
	return blake3.Sum256(data)
}

// SumReader computes the BLAKE3 hash of the data from the reader
func SumReader(r io.Reader) (Hash, error) {
	h := blake3.New()
	if _, err := io.Copy(h, r); err != nil {
		return Hash{}, err
	}
	var out Hash
	copy(out[:], h.Sum(nil))
	return out, nil
}

// Parse converts a hex string to a Hash
func Parse(s string) (Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, err
	}
	var h Hash
	copy(h[:], b)
	return h, nil
}
