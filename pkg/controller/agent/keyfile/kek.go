package keyfile

import "golang.org/x/crypto/argon2"

// Key sizing for the agent backend.
//
// The data key is a 32-byte (AES-256) key generated on the first unlock, when the
// user sets the passphrase. It is wrapped with a key-encryption key (KEK) derived
// from that passphrase via Argon2id.
const (
	dataKeyLen = 32 // AES-256 key length in bytes
	saltLen    = 16 // Argon2id salt length in bytes

	// Argon2id cost parameters. These are paid on each `ghtkn agent unlock` that
	// reaches the key file, i.e. normally once per agent start.
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

// zero overwrites b with zeros. It is used to scrub the derived KEK after it has
// wrapped or unwrapped the data key, so the passphrase-derived key does not linger.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
