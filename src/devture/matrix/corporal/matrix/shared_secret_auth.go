package matrix

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
)

type SharedSecretAuthPasswordGenerator struct {
	sharedSecret string
}

func NewSharedSecretAuthPasswordGenerator(sharedSecret string) *SharedSecretAuthPasswordGenerator {
	return &SharedSecretAuthPasswordGenerator{
		sharedSecret: sharedSecret,
	}
}

func (me *SharedSecretAuthPasswordGenerator) GenerateForUserId(userId string) string {
	//We expect the server to be running with the SharedSecretAuthenticator
	//password provider which, if configured correctly, will understand our "fake" passwords.
	m := hmac.New(sha512.New, []byte(me.sharedSecret))
	m.Write([]byte(userId))
	return fmt.Sprintf("%x", m.Sum(nil))
}
