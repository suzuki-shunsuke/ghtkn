package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Cryptographic parameters for the agent backend.
//
// The data key is a 32-byte (AES-256) key generated on first start. It is wrapped
// with a key-encryption key (KEK) derived from the user's passphrase via Argon2id.
const (
	dataKeyLen = 32 // AES-256 key length in bytes
	saltLen    = 16 // Argon2id salt length in bytes

	// Argon2id cost parameters. These are paid once per `ghtkn agent start`.
	// 64 MiB / time=3 / parallelism=4 is a common desktop-grade default that
	// comfortably exceeds the OWASP minimum (19 MiB, time=2).
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // memory in KiB (= 64 MiB)
	argon2Threads = 4
)

// errDecrypt is returned by open when the ciphertext fails authentication,
// which typically means a wrong key (e.g. an incorrect passphrase) or corruption.
var errDecrypt = errors.New("decrypt: authentication failed")

// deriveKEK derives a 32-byte key-encryption key from a passphrase and salt
// using Argon2id. The same passphrase and salt always yield the same key.
func deriveKEK(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argon2Time, argon2Memory, argon2Threads, dataKeyLen)
}

// seal encrypts plaintext with AES-256-GCM using key and returns nonce||ciphertext.
// A fresh random nonce is generated for every call.
func seal(key, plaintext []byte) ([]byte, error) {
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

// open decrypts a nonce||ciphertext blob produced by seal with the given key.
// It returns errDecrypt when authentication fails.
func open(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, errDecrypt
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errDecrypt
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
