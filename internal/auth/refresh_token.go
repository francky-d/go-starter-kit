package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const refreshTokenBytes = 32

// NewRefreshToken returns a random token for the client and the hash to
// store. Only the hash is kept, so a leaked table exposes no usable token.
func NewRefreshToken() (token, hash string) {
	random := make([]byte, refreshTokenBytes)
	_, _ = rand.Read(random) // never returns an error: it crashes the program instead

	token = base64.RawURLEncoding.EncodeToString(random)

	return token, HashRefreshToken(token)
}

// HashRefreshToken needs no salt or slow hash: the token is 256 random bits,
// so it cannot be guessed from its hash the way a password can.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}
