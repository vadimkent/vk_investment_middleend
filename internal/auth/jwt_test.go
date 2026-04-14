package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

func mintToken(t *testing.T, sub string, iat, exp time.Time, secret string) string {
	t.Helper()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]any{"sub": sub, "iat": iat.Unix(), "exp": exp.Unix()}

	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := enc.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig
}

func TestValidate_ValidToken(t *testing.T) {
	now := time.Now()
	tok := mintToken(t, "user-123", now, now.Add(1*time.Hour), testSecret)

	uid, err := Validate(tok, testSecret, 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "user-123", uid)
}

func TestValidate_Expired(t *testing.T) {
	now := time.Now()
	tok := mintToken(t, "user-123", now.Add(-2*time.Hour), now.Add(-1*time.Hour), testSecret)

	_, err := Validate(tok, testSecret, 30*time.Second)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestValidate_ExpiredWithinLeeway(t *testing.T) {
	now := time.Now()
	tok := mintToken(t, "user-123", now.Add(-1*time.Hour), now.Add(-10*time.Second), testSecret)

	uid, err := Validate(tok, testSecret, 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "user-123", uid)
}

func TestValidate_BadSignature(t *testing.T) {
	now := time.Now()
	tok := mintToken(t, "user-123", now, now.Add(1*time.Hour), "wrong-secret")

	_, err := Validate(tok, testSecret, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidate_Malformed(t *testing.T) {
	for _, tok := range []string{"", "abc", "a.b", "a.b.c.d"} {
		_, err := Validate(tok, testSecret, 0)
		require.Error(t, err, "input=%q", tok)
		assert.ErrorIs(t, err, ErrMalformedToken, "input=%q", tok)
	}
}

func TestValidate_WrongAlgorithm(t *testing.T) {
	enc := base64.RawURLEncoding
	header, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{"sub": "u", "exp": time.Now().Add(1 * time.Hour).Unix()})
	tok := enc.EncodeToString(header) + "." + enc.EncodeToString(payload) + "."

	_, err := Validate(tok, testSecret, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAlgorithm)
}

func TestValidate_MissingSub(t *testing.T) {
	now := time.Now()
	tok := mintToken(t, "", now, now.Add(1*time.Hour), testSecret)

	_, err := Validate(tok, testSecret, 0)
	require.Error(t, err)
}
