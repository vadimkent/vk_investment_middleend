package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrMalformedToken   = errors.New("malformed token")
	ErrInvalidAlgorithm = errors.New("invalid algorithm")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTokenExpired     = errors.New("token expired")
	ErrMissingSubject   = errors.New("missing subject")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// Validate parses an HS256 JWT, verifies its signature with secret, checks exp
// with the given leeway, and returns the sub claim.
func Validate(token, secret string, leeway time.Duration) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return "", ErrMalformedToken
	}

	enc := base64.RawURLEncoding

	hb, err := enc.DecodeString(parts[0])
	if err != nil {
		return "", ErrMalformedToken
	}
	var h header
	if err := json.Unmarshal(hb, &h); err != nil {
		return "", ErrMalformedToken
	}
	if h.Alg != "HS256" {
		return "", ErrInvalidAlgorithm
	}

	pb, err := enc.DecodeString(parts[1])
	if err != nil {
		return "", ErrMalformedToken
	}
	var cl claims
	if err := json.Unmarshal(pb, &cl); err != nil {
		return "", ErrMalformedToken
	}

	if parts[2] == "" {
		return "", ErrMalformedToken
	}

	sig, err := enc.DecodeString(parts[2])
	if err != nil {
		return "", ErrMalformedToken
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return "", ErrInvalidSignature
	}

	if cl.Exp > 0 && time.Now().Add(-leeway).Unix() > cl.Exp {
		return "", ErrTokenExpired
	}

	if cl.Sub == "" {
		return "", ErrMissingSubject
	}
	return cl.Sub, nil
}
