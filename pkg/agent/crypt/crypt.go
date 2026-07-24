// Package crypt provides the agent's low-level at-rest primitives: AES-256-GCM
// encryption (Seal/Open) and an atomic file write. It carries no domain knowledge of
// keys or tokens; the keyfile and tokenstore packages build on it.
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// ErrDecrypt is returned by Open when the ciphertext fails authentication, which
// typically means a wrong key (e.g. an incorrect passphrase) or corruption.
var ErrDecrypt = errors.New("decrypt: authentication failed")

// Seal encrypts plaintext with AES-256-GCM using key and returns nonce||ciphertext.
// A fresh random nonce is generated for every call.
func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate a nonce: %w", err)
	}
	// Seal appends the ciphertext to nonce, yielding nonce||ciphertext.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decrypts a nonce||ciphertext blob produced by Seal with the given key.
// It returns ErrDecrypt when authentication fails.
func Open(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, ErrDecrypt
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

// newGCM builds an AES-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create an AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create a GCM cipher: %w", err)
	}
	return gcm, nil
}
