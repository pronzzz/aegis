package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

const (
	KeySize   = 32
	NonceSize = 12
)

// MasterKey represents the 256-bit repository key
type MasterKey [KeySize]byte

// NewMasterKey generates a random master key
func NewMasterKey() (MasterKey, error) {
	var k MasterKey
	if _, err := io.ReadFull(rand.Reader, k[:]); err != nil {
		return k, err
	}
	return k, nil
}

// DeriveKeyFromPassphrase derives a key using Argon2id
// salt must be 16 bytes
func DeriveKeyFromPassphrase(passphrase string, salt []byte) []byte {
	// Params: time=1, memory=64MB, threads=4, keyLen=32
	return argon2.IDKey([]byte(passphrase), salt, 1, 64*1024, 4, 32)
}

// Encrypt encrypts data using AES-256-GCM with a random nonce.
// The nonce is prepended to the ciphertext.
func (k MasterKey) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using AES-256-GCM.
// Expects nonce prepended to ciphertext.
func (k MasterKey) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, encryptedData, nil)
}

// KeyFile structure for storing the encrypted master key
type KeyFile struct {
	Salt         []byte `json:"salt"`
	EncryptedKey []byte `json:"encrypted_key"`
	Algorithm    string `json:"algo"` // "argon2id_aes256gcm"
}

// SaveKey stores the master key to disk, encrypted by the passphrase
func SaveKey(path string, mk MasterKey, passphrase string) error {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	kekFunc := DeriveKeyFromPassphrase(passphrase, salt)
	var kek MasterKey
	copy(kek[:], kekFunc)

	encryptedMK, err := kek.Encrypt(mk[:])
	if err != nil {
		return err
	}

	kf := KeyFile{
		Salt:         salt,
		EncryptedKey: encryptedMK,
		Algorithm:    "argon2id_aes256gcm",
	}

	data, err := json.Marshal(kf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LoadKey loads the master key from disk using the passphrase
func LoadKey(path string, passphrase string) (MasterKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MasterKey{}, err
	}

	var kf KeyFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return MasterKey{}, err
	}

	kekFunc := DeriveKeyFromPassphrase(passphrase, kf.Salt)
	var kek MasterKey
	copy(kek[:], kekFunc)

	decryptedBytes, err := kek.Decrypt(kf.EncryptedKey)
	if err != nil {
		return MasterKey{}, fmt.Errorf("invalid passphrase or corrupted key file")
	}

	var mk MasterKey
	copy(mk[:], decryptedBytes)
	return mk, nil
}
