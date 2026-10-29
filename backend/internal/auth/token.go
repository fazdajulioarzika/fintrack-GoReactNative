// Package auth berisi pembuatan dan verifikasi token.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// NewAccessToken membuat JWT berumur pendek. Isinya hanya ID user (subject).
func NewAccessToken(userID, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken memverifikasi tanda tangan + masa berlaku, lalu mengembalikan user ID.
func ParseAccessToken(tokenStr, secret string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"})) // tolak algoritma lain (mis. "none")
	if err != nil || !token.Valid {
		return "", errors.New("token tidak valid")
	}
	if claims.Subject == "" {
		return "", errors.New("token tidak valid")
	}
	return claims.Subject, nil
}

// NewRefreshToken membuat token acak. `raw` dikirim ke klien, `hash` disimpan di database.
func NewRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefreshToken(raw), nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
