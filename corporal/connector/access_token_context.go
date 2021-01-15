package connector

import (
	"sync"
	"time"
)

type AccessTokenContext struct {
	connector       MatrixConnector
	deviceId        string
	validitySeconds int

	userIdToAccessTokenMap *sync.Map
}

func NewAccessTokenContext(connector MatrixConnector, deviceId string, validitySeconds int) *AccessTokenContext {
	return &AccessTokenContext{
		connector:       connector,
		deviceId:        deviceId,
		validitySeconds: validitySeconds,

		userIdToAccessTokenMap: &sync.Map{},
	}
}

func (me *AccessTokenContext) GetAccessTokenForUserId(userId string) (string, error) {
	accessTokenInterface, ok := me.userIdToAccessTokenMap.Load(userId)
	if ok {
		accessToken := accessTokenInterface.(*AccessToken)

		if !accessToken.Expired() {
			// Well, it hasn't expired, but may be expiring soon (say, in 1 second), which could be problematic.
			// We don't handle this edge-case for the time being.
			return accessToken.Token(), nil
		}

		// We may wish to destroy the expired token, but we can't. We get this on every API call:
		// > {"errcode":"M_UNKNOWN_TOKEN","error":"Access token has expired","soft_logout":true}
		//
		// So, we can only forget about it and proceed to getting a new one.

		me.ClearAccessTokenForUserId(userId)
	}

	var validUntil *time.Time
	if me.validitySeconds != 0 {
		validUntilT := time.Now().Add(time.Duration(me.validitySeconds) * time.Second)
		validUntil = &validUntilT
	}

	accessTokenString, err := me.connector.ObtainNewAccessTokenForUserId(userId, me.deviceId, validUntil)
	if err != nil {
		return "", err
	}

	// The first time we obtain a token, let's verify it works and belongs to the user we expect.
	err = me.connector.VerifyAccessToken(userId, accessTokenString)
	if err != nil {
		return "", err
	}

	accessToken := newAccessToken(accessTokenString, validUntil)

	me.userIdToAccessTokenMap.Store(userId, accessToken)

	return accessTokenString, nil
}

func (me *AccessTokenContext) ClearAccessTokenForUserId(userId string) {
	me.userIdToAccessTokenMap.Delete(userId)
}

func (me *AccessTokenContext) Release() {
	me.userIdToAccessTokenMap.Range(func(userId interface{}, accessTokenInterface interface{}) bool {
		accessToken := accessTokenInterface.(*AccessToken)

		me.connector.DestroyAccessToken(userId.(string), accessToken.Token())

		me.userIdToAccessTokenMap.Delete(userId)

		return true
	})
}
