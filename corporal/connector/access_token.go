package connector

import "time"

// AccessToken represents an obtained access token and some metadata about it (validity, etc.)
type AccessToken struct {
	token      string
	validUntil *time.Time
}

func newAccessToken(token string, validUntil *time.Time) *AccessToken {
	return &AccessToken{
		token:      token,
		validUntil: validUntil,
	}
}

func (me AccessToken) Token() string {
	return me.token
}

func (me AccessToken) Expired() bool {
	if me.validUntil == nil {
		return false
	}

	return time.Now().Unix() > (*me.validUntil).Unix()
}
