package userauth

// NoneAuthenticator is a user authenticator which outright-rejects all authentication attempts.
//
// It's useful to use (but not a requirement) as an authentication type for users which are not active.
// Inactive users do not go through an authenticator (we reject those early either way),
// so it doesn't matter which authentication type is used for them.
// Still, it makes sense to clear away all credentials for such accounts.
type NoneAuthenticator struct {
}

func NewNoneAuthenticator() *NoneAuthenticator {
	return &NoneAuthenticator{}
}

func (me *NoneAuthenticator) Type() string {
	return "none"
}

func (me *NoneAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	return false, nil
}
