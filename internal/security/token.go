package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/farkhanisturkia/gohan/internal/config"
)

func GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HashToken(rawToken string) string {
	env := config.GetEnv()
	secret := env.AppKey

	if secret == "" {
		secret = "gohan-fallback-secret-key"
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(rawToken))
	return hex.EncodeToString(h.Sum(nil))
}