package storage

import "io"

// Backend defines the interface for physical storage systems (Local, S3, etc.)
type Backend interface {
	// Put stores the data with the given key (hash)
	Put(key string, data []byte) error

	// Get retrieves the data for the given key
	Get(key string) ([]byte, error)

	// Has checks if the key exists
	Has(key string) (bool, error)

	// Close releases any resources
	Close() error
}

// Reader is an optional interface for backends that support streaming read
type Reader interface {
	GetReader(key string) (io.ReadCloser, error)
}
