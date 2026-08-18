package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID any    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID any, email string, secret string, duration any) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret key cannot be empty")
	}

	var expiryTime time.Duration

	switch v := duration.(type) {
	case time.Duration:
		expiryTime = v
	case string:
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return "", fmt.Errorf("invalid duration format: %w", err)
		}
		expiryTime = parsed
	default:
		return "", errors.New("duration must be time.Duration or string format (e.g. '24h')")
	}

	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString string, secret string) (*JWTClaims, error) {
	if secret == "" {
		return nil, errors.New("JWT secret key cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid JWT token")
}