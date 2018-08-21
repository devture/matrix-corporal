package userauth

type Authenticator interface {
	Type() string
	Authenticate(userId, givenPassword, authCredential string) (bool, error)
}
