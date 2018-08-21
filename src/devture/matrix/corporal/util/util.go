package util

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
)

func IsStringInArray(needle string, haystack []string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

func Sha512(value string) string {
	m := sha512.New()
	m.Write([]byte(value))
	return fmt.Sprintf("%x", m.Sum(nil))
}
