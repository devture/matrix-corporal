package userauth

import (
	"crypto/subtle"
)

// PlainAuthenticator is a user authenticator using plain-text credentials.
type PlainAuthenticator struct {
}

func NewPlainAuthenticator() *PlainAuthenticator {
	return &PlainAuthenticator{}
}

func (me *PlainAuthenticator) Type() string {
	return "plain"
}

func (me *PlainAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	return subtle.ConstantTimeCompare([]byte(givenPassword), []byte(authCredential)) == 1, nil
}
