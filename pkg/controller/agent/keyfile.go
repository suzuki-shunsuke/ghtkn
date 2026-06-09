package agent

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
)

// Key file layout: version(1) || salt(saltLen) || wrapped data key (nonce||ciphertext).
// The wrapped data key is the 32-byte data key encrypted with the passphrase-derived
// KEK using AES-256-GCM.
const (
	keyFileVersion    = 1
	keyFilePerm       = tokenFilePerm
	keyFileHeaderSize = 1 + saltLen // version byte + salt
)

// errIncorrectPassphrase is returned when the key file cannot be unwrapped with the
// supplied passphrase, which means the passphrase is wrong (or the file is corrupt).
var errIncorrectPassphrase = errors.New("incorrect passphrase")

// loadOrCreateDataKey loads the data key from path, decrypting it with passphrase.
// If the file does not exist, it generates a new data key, wraps it with a
// passphrase-derived KEK, writes the key file (0600), and returns the data key.
// The bool result reports whether a new key file was created.
func loadOrCreateDataKey(path string, passphrase []byte) ([]byte, bool, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dataKey, cerr := createDataKey(path, passphrase)
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

// createDataKey generates a new random data key and salt, wraps the data key with
// the passphrase-derived KEK, and writes the key file atomically.
func createDataKey(path string, passphrase []byte) ([]byte, error) {
	dataKey := make([]byte, dataKeyLen)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, fmt.Errorf("generate a data key: %w", err)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate a salt: %w", err)
	}
	wrapped, err := seal(deriveKEK(passphrase, salt), dataKey)
	if err != nil {
		return nil, fmt.Errorf("wrap the data key: %w", err)
	}
	blob := make([]byte, 0, keyFileHeaderSize+len(wrapped))
	blob = append(blob, keyFileVersion)
	blob = append(blob, salt...)
	blob = append(blob, wrapped...)
	if err := atomicWrite(path, blob); err != nil {
		return nil, fmt.Errorf("write the key file: %w", err)
	}
	return dataKey, nil
}

// unwrapDataKey parses a key file blob and decrypts the data key with passphrase.
// It returns errIncorrectPassphrase when decryption fails.
func unwrapDataKey(blob, passphrase []byte) ([]byte, error) {
	if len(blob) < keyFileHeaderSize {
		return nil, errors.New("the key file is too short")
	}
	if blob[0] != keyFileVersion {
		return nil, fmt.Errorf("unsupported key file version: %d", blob[0])
	}
	salt := blob[1:keyFileHeaderSize]
	wrapped := blob[keyFileHeaderSize:]
	dataKey, err := open(deriveKEK(passphrase, salt), wrapped)
	if err != nil {
		if errors.Is(err, errDecrypt) {
			return nil, errIncorrectPassphrase
		}
		return nil, fmt.Errorf("unwrap the data key: %w", err)
	}
	return dataKey, nil
}
