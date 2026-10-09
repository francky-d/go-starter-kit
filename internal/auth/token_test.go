package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/config"
)

const secret = "a-test-secret-that-is-at-least-32-bytes-long"

func newIssuer() *auth.TokenIssuer {
	return auth.NewTokenIssuer(config.Auth{
		JWTSecret:           secret,
		JWTIssuer:           "go-starter-kit",
		AccessTokenLifetime: 15 * time.Minute,
	})
}

func at(moment time.Time) func() time.Time {
	return func() time.Time { return moment }
}

func TestTokenIssuer_IssuesATokenThatIdentifiesTheUser(t *testing.T) {
	issuer := newIssuer()
	userID := uuid.Must(uuid.NewV7())

	token, err := issuer.Issue(userID)
	require.NoError(t, err)
	identified, err := issuer.UserID(token.Value)

	require.NoError(t, err)
	assert.Equal(t, userID, identified)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), token.ExpiresAt, time.Second)
}

func TestTokenIssuer_RefusesAnExpiredToken(t *testing.T) {
	issuedAt := time.Now()
	token, err := newIssuer().WithClock(at(issuedAt)).Issue(uuid.Must(uuid.NewV7()))
	require.NoError(t, err)

	_, err = newIssuer().WithClock(at(issuedAt.Add(16 * time.Minute))).UserID(token.Value)

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestTokenIssuer_ToleratesSmallClockDifferences(t *testing.T) {
	issuedAt := time.Now()
	token, err := newIssuer().WithClock(at(issuedAt)).Issue(uuid.Must(uuid.NewV7()))
	require.NoError(t, err)

	_, err = newIssuer().WithClock(at(issuedAt.Add(15*time.Minute + 10*time.Second))).UserID(token.Value)

	assert.NoError(t, err)
}

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)

	return token
}

// alterSignature changes the first character of the signature. The last
// one could differ only in padding bits and decode to the same bytes.
func alterSignature(token string) string {
	signatureStart := strings.LastIndex(token, ".") + 1
	replacement := "A"
	if token[signatureStart] == 'A' {
		replacement = "B"
	}

	return token[:signatureStart] + replacement + token[signatureStart+1:]
}

func validClaims() jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   uuid.Must(uuid.NewV7()).String(),
		Issuer:    "go-starter-kit",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}
}

func TestTokenIssuer_RefusesForgedTokens(t *testing.T) {
	withoutExpiry := validClaims()
	withoutExpiry.ExpiresAt = nil
	otherIssuer := validClaims()
	otherIssuer.Issuer = "someone-else"
	notAUser := validClaims()
	notAUser.Subject = "admin"
	issuedInTheFuture := validClaims()
	issuedInTheFuture.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
	notYetValid := validClaims()
	notYetValid.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Minute))

	cases := map[string]string{
		"signed with another secret": sign(t, jwt.SigningMethodHS256, []byte("another-secret-that-is-32-bytes-long!!"), validClaims()),
		"signed with HS512":          sign(t, jwt.SigningMethodHS512, []byte(secret), validClaims()),
		"unsigned":                   sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"without expiry":             sign(t, jwt.SigningMethodHS256, []byte(secret), withoutExpiry),
		"from another issuer":        sign(t, jwt.SigningMethodHS256, []byte(secret), otherIssuer),
		"subject is not a user id":   sign(t, jwt.SigningMethodHS256, []byte(secret), notAUser),
		"issued in the future":       sign(t, jwt.SigningMethodHS256, []byte(secret), issuedInTheFuture),
		"not valid yet":              sign(t, jwt.SigningMethodHS256, []byte(secret), notYetValid),
		"with an altered signature":  alterSignature(sign(t, jwt.SigningMethodHS256, []byte(secret), validClaims())),
		"not a token":                "not-a-jwt",
		"empty":                      "",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newIssuer().UserID(token)

			assert.ErrorIs(t, err, auth.ErrInvalidToken)
		})
	}
}
