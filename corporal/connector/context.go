package connector

import (
	"sync"
)

type AccessTokenContext struct {
	connector MatrixConnector
	deviceId  string

	userIdToAccessTokenMap *sync.Map
}

func NewAccessTokenContext(connector MatrixConnector, deviceId string) *AccessTokenContext {
	return &AccessTokenContext{
		connector: connector,
		deviceId:  deviceId,

		userIdToAccessTokenMap: &sync.Map{},
	}
}

func (me *AccessTokenContext) GetAccessTokenForUserId(userId string) (string, error) {
	accessTokenInterface, ok := me.userIdToAccessTokenMap.Load(userId)
	if ok {
		return accessTokenInterface.(string), nil
	}

	accessToken, err := me.connector.ObtainNewAccessTokenForUserId(userId, me.deviceId)
	if err != nil {
		return "", err
	}

	// The first time we obtain a token, let's verify it works and belongs to the user we expect.
	err = me.connector.VerifyAccessToken(userId, accessToken)
	if err != nil {
		return "", err
	}

	me.userIdToAccessTokenMap.Store(userId, accessToken)

	return accessToken, nil
}

func (me *AccessTokenContext) ClearAccessTokenForUserId(userId string) {
	me.userIdToAccessTokenMap.Delete(userId)
}

func (me *AccessTokenContext) Release() {
	me.userIdToAccessTokenMap.Range(func(userId interface{}, accessToken interface{}) bool {
		me.connector.DestroyAccessToken(userId.(string), accessToken.(string))
		me.userIdToAccessTokenMap.Delete(userId)
		return true
	})
}
