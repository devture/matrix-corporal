package userauth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type BcryptAuthenticator struct {
}

func NewBcryptAuthenticator() *BcryptAuthenticator {
	return &BcryptAuthenticator{}
}

func (me *BcryptAuthenticator) Type() string {
	return "bcrypt"
}

func (me *BcryptAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	if len(givenPassword) > 4096 {
		// To avoid a DoS, avoid dealing with too long inputs.
		return false, fmt.Errorf("Rejecting long password (%d)", len(givenPassword))
	}

	err := bcrypt.CompareHashAndPassword([]byte(authCredential), []byte(givenPassword))
	if err != nil {
		return false, err
	}

	return true, nil
}
