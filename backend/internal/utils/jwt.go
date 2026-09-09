package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// NOTE: This is a stateless JWT implementation — a token cannot be looked up
// and torn down individually. Revocation is handled with a token_version claim
// that is compared against the user's current version on every request, so a
// password reset invalidates every token issued before it.

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrNoJWTSecret  = errors.New("JWT_SECRET environment variable is not set")
)

// TokenTTL is how long an access token stays valid.
const TokenTTL = 24 * time.Hour

// Claims is the typed claim set. Using a struct rather than jwt.MapClaims means
// a missing or wrongly-typed field is a parse error instead of a silent zero.
type Claims struct {
	UserID       int `json:"user_id"`
	TokenVersion int `json:"token_version"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrNoJWTSecret
	}
	return []byte(secret), nil
}

func GenerateToken(userID, tokenVersion int) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ValidateToken verifies the signature and the standard time claims, and
// requires exp to be present — jwt.Parse only checks exp when it exists, so
// without WithExpirationRequired a token that simply omits it never expires.
func ValidateToken(tokenStr string) (userID int, tokenVersion int, err error) {
	secret, err := jwtSecret()
	if err != nil {
		return 0, 0, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return 0, 0, ErrInvalidToken
	}
	if claims.UserID <= 0 {
		return 0, 0, ErrInvalidToken
	}

	return claims.UserID, claims.TokenVersion, nil
}
