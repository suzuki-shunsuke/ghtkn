// Package keyfile manages the agent's data key on disk: a 32-byte AES-256 data key
// wrapped with a passphrase-derived (Argon2id) key-encryption key and stored as a key
// file. It encrypts/decrypts via the crypt package and resolves the key file path.
package keyfile

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/crypt"
)

// Key file layout: version(1) || salt(saltLen) || wrapped data key (nonce||ciphertext).
// The wrapped data key is the 32-byte data key encrypted with the passphrase-derived
// KEK using AES-256-GCM.
const (
	keyFileVersion                = 1
	keyFilePerm       os.FileMode = 0o600       // matches crypt.AtomicWrite
	keyFileHeaderSize             = 1 + saltLen // version byte + salt
)

// ErrIncorrectPassphrase is returned when the key file cannot be unwrapped with the
// supplied passphrase, which means the passphrase is wrong (or the file is corrupt).
var ErrIncorrectPassphrase = errors.New("incorrect passphrase")

// LoadOrCreateDataKey loads the data key from path, decrypting it with passphrase.
// If the file does not exist, it generates a new data key, wraps it with a
// passphrase-derived KEK, writes the key file (0600), and returns the data key.
// The bool result reports whether a new key file was created.
func LoadOrCreateDataKey(path string, passphrase []byte) ([]byte, bool, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dataKey, cerr := CreateDataKey(path, passphrase)
			return dataKey, true, cerr
		}
		return nil, false, fmt.Errorf("read the key file: %w", err)
	}
	dataKey, err := unwrapDataKey(blob, passphrase)
	if err != nil {
		return nil, false, err
	}
	return dataKey, false, nil
}

// CreateDataKey generates a new random data key and salt, wraps the data key with
// the passphrase-derived KEK, and writes the key file atomically.
func CreateDataKey(path string, passphrase []byte) ([]byte, error) {
	dataKey := make([]byte, dataKeyLen)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("generate a data key: %w", err)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate a salt: %w", err)
	}
	kek := deriveKEK(passphrase, salt)
	defer zero(kek) // the KEK is only needed to wrap the data key; do not keep it in memory
	wrapped, err := crypt.Seal(kek, dataKey)
	if err != nil {
		return nil, fmt.Errorf("wrap the data key: %w", err)
	}
	blob := make([]byte, 0, keyFileHeaderSize+len(wrapped))
	blob = append(blob, keyFileVersion)
	blob = append(blob, salt...)
	blob = append(blob, wrapped...)
	if err := crypt.AtomicWrite(path, blob); err != nil {
		return nil, fmt.Errorf("write the key file: %w", err)
	}
	return dataKey, nil
}

// unwrapDataKey parses a key file blob and decrypts the data key with passphrase.
// It returns ErrIncorrectPassphrase when decryption fails.
func unwrapDataKey(blob, passphrase []byte) ([]byte, error) {
	if len(blob) < keyFileHeaderSize {
		return nil, errors.New("the key file is too short")
	}
	if blob[0] != keyFileVersion {
		return nil, fmt.Errorf("unsupported key file version: %d", blob[0])
	}
	salt := blob[1:keyFileHeaderSize]
	wrapped := blob[keyFileHeaderSize:]
	kek := deriveKEK(passphrase, salt)
	defer zero(kek) // the KEK is only needed to unwrap the data key; do not keep it in memory
	dataKey, err := crypt.Open(kek, wrapped)
	if err != nil {
		if errors.Is(err, crypt.ErrDecrypt) {
			return nil, ErrIncorrectPassphrase
		}
		return nil, fmt.Errorf("unwrap the data key: %w", err)
	}
	return dataKey, nil
}
