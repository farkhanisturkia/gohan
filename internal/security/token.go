package security

import (
	"fmt"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/farkhanisturkia/gohan/internal/config"
)

func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HashToken(rawToken string) (string, error) {
    env := config.GetEnv()
    secret := env.AppKey

    if secret == "" {
        return "", fmt.Errorf("APP_KEY is required for token hashing")
    }

    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(rawToken))
    return hex.EncodeToString(h.Sum(nil)), nil
}