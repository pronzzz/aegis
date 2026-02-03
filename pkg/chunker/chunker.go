package chunker

import (
	"io"

	"github.com/pranavdwivedi/aegis/pkg/hash"
)

// DefaultChunkSize is 4MB (4 * 1024 * 1024 bytes)
const DefaultChunkSize = 4 * 1024 * 1024

// Chunk represents a piece of a file
type Chunk struct {
	Data []byte
	Hash hash.Hash
}

// Chunker is the interface for splitting data into chunks
type Chunker interface {
	Next() (*Chunk, error)
}

// FixedSizeChunker implements the Chunker interface with fixed-size blocks
type FixedSizeChunker struct {
	reader io.Reader
	buf    []byte
}

// NewFixedSizeChunker creates a new FixedSizeChunker
func NewFixedSizeChunker(r io.Reader, size int) *FixedSizeChunker {
	if size <= 0 {
		size = DefaultChunkSize
	}
	return &FixedSizeChunker{
		reader: r,
		buf:    make([]byte, size),
	}
}

// Next reads the next chunk from the underlying reader
func (c *FixedSizeChunker) Next() (*Chunk, error) {
	n, err := io.ReadFull(c.reader, c.buf)
	if n == 0 {
		if err == io.EOF {
			return nil, io.EOF
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
	}

	// If we read less than the buffer size, this is the last chunk
	data := make([]byte, n)
	copy(data, c.buf[:n])

	// Compute hash
	h := hash.Sum(data)

	// If io.ReadFull returns ErrUnexpectedEOF, it just means we hit EOF in the middle of a chunk.
	// For us, that's just a valid partial chunk.
	// If it returned standard EOF with n=0, we handled it above.
	if err == io.ErrUnexpectedEOF {
		err = nil
	}

	return &Chunk{
		Data: data,
		Hash: h,
	}, err
}
