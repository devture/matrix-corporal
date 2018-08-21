package userauth

import "fmt"

type Checker struct {
	authenticators map[string]Authenticator
}

func NewChecker() *Checker {
	return &Checker{
		authenticators: map[string]Authenticator{},
	}
}

func (me *Checker) RegisterAuthenticator(entity Authenticator) {
	me.authenticators[entity.Type()] = entity
}

func (me *Checker) Check(userId, givenPassword, authType, authCredential string) (bool, error) {
	authenticator, ok := me.authenticators[authType]
	if !ok {
		return false, fmt.Errorf("Unsupported authenticator: %s", authType)
	}

	return authenticator.Authenticate(userId, givenPassword, authCredential)
}
