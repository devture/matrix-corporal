package userauth

import "devture-matrix-corporal/corporal/util"

type Authenticator interface {
	Type() string
	Authenticate(userId, givenPassword, authCredential string) (bool, error)
}

const (
	UserAuthTypePlain       = "plain"
	UserAuthTypePassthrough = "passthrough"
	UserAuthTypeMd5         = "md5"
	UserAuthTypeSha1        = "sha1"
	UserAuthTypeSha256      = "sha256"
	UserAuthTypeSha512      = "sha512"
	UserAuthTypeBcrypt      = "bcrypt"
	UserAuthTypeREST        = "rest"
)

var knownUserAuthTypes = []string{
	UserAuthTypePlain,
	UserAuthTypePassthrough,
	UserAuthTypeMd5,
	UserAuthTypeSha1,
	UserAuthTypeSha256,
	UserAuthTypeSha512,
	UserAuthTypeBcrypt,
	UserAuthTypeREST,
}

func IsKnownUserAuthType(value string) bool {
	return util.IsStringInArray(value, knownUserAuthTypes)
}
