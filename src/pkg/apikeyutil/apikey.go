package apikeyutil

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "fmt"
)

const defaultEntropyBytes = 32

// GeneratePlainKey returns a randomly generated API key using crypto/rand. The returned value includes the
// configured prefix if provided and is suitable for sharing with customers exactly once.
func GeneratePlainKey(prefix string, entropyBytes int) (string, string, error) {
    if entropyBytes <= 0 {
        entropyBytes = defaultEntropyBytes
    }

    buffer := make([]byte, entropyBytes)
    if _, err := rand.Read(buffer); err != nil {
        return "", "", fmt.Errorf("failed to read crypto entropy: %w", err)
    }

    randomPart := base64.RawURLEncoding.EncodeToString(buffer)
    if prefix != "" {
        return fmt.Sprintf("%s_%s", prefix, randomPart), randomPart[:min(8, len(randomPart))], nil
    }
    return randomPart, randomPart[:min(8, len(randomPart))], nil
}

// HashKey hashes the provided API key using HMAC-SHA256 with the configured salt so we never persist the
// raw secret in the management database.
func HashKey(rawKey, salt string) (string, error) {
    if salt == "" {
        return "", fmt.Errorf("api key salt is empty")
    }

    mac := hmac.New(sha256.New, []byte(salt))
    if _, err := mac.Write([]byte(rawKey)); err != nil {
        return "", fmt.Errorf("failed to hash api key: %w", err)
    }

    return hex.EncodeToString(mac.Sum(nil)), nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
