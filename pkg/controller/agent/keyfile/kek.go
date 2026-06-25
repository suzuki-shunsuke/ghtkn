package keyfile

import "golang.org/x/crypto/argon2"

// Key sizing for the agent backend.
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

// deriveKEK derives a 32-byte key-encryption key from a passphrase and salt
// using Argon2id. The same passphrase and salt always yield the same key.
func deriveKEK(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argon2Time, argon2Memory, argon2Threads, dataKeyLen)
}
