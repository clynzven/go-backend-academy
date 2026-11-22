package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id settings from the OWASP password storage cheat sheet:
// 19 MiB of memory, 2 passes, 1 thread. Every guess costs the same.
const (
	argonTime    = 2
	argonMemory  = 19 * 1024 // KiB
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
)

// HashPassword returns a self-describing argon2id hash:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
func HashPassword(password string) string {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

// checkPassword reports whether password matches stored, and whether stored
// is an old-style hash that should be replaced now that we know the password.
func checkPassword(password, stored string) (ok, upgrade bool) {
	if !strings.HasPrefix(stored, "$argon2id$") {
		// Lesson 2's plain SHA-256 (no salt, one fast round): still accepted once,
		// then upgraded to argon2id.
		sum := sha256.Sum256([]byte(password))
		return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(stored)) == 1, true
	}
	var version, memory, time, threads int
	var saltB64, keyB64 string
	parts := strings.Split(stored, "$") // "", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash
	if len(parts) != 6 {
		return false, false
	}
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, false
	}
	saltB64, keyB64 = parts[4], parts[5]
	salt, err1 := base64.RawStdEncoding.DecodeString(saltB64)
	want, err2 := base64.RawStdEncoding.DecodeString(keyB64)
	if err1 != nil || err2 != nil {
		return false, false
	}
	got := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, false
}

// dummyHash is checked when the email doesn't exist, so a wrong email takes
// as long as a wrong password: response times don't reveal who has an account.
var dummyHash = HashPassword("not a real password")
