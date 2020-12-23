package userauth

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"hash"
)

// HashAuthenticator is a user authenticator using hashed credentials (like md5, sha1, sha256, sha512).
type HashAuthenticator struct {
	hash     hash.Hash
	authType string
}

func NewHashAuthenticator(hash hash.Hash, authType string) *HashAuthenticator {
	return &HashAuthenticator{
		hash:     hash,
		authType: authType,
	}
}

func (me *HashAuthenticator) Type() string {
	return me.authType
}

func (me *HashAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	authCredentialBytes, err := hex.DecodeString(authCredential)
	if err != nil {
		return false, err
	}

	me.hash.Reset()
	me.hash.Write([]byte(givenPassword))
	hashBytes := me.hash.Sum(nil)

	return subtle.ConstantTimeCompare(authCredentialBytes, hashBytes) == 1, nil
}

func NewMd5Authenticator() Authenticator {
	return NewHashAuthenticator(md5.New(), UserAuthTypeMd5)
}

func NewSha1Authenticator() Authenticator {
	return NewHashAuthenticator(sha1.New(), UserAuthTypeSha1)
}

func NewSha256Authenticator() Authenticator {
	return NewHashAuthenticator(sha256.New(), UserAuthTypeSha256)
}

func NewSha512Authenticator() Authenticator {
	return NewHashAuthenticator(sha512.New(), UserAuthTypeSha512)
}
