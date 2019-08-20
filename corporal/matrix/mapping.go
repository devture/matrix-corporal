package matrix

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"

	"github.com/matrix-org/gomatrix"
	"github.com/sirupsen/logrus"
)

// userIdUnknownToken is a special mapping value for when the access token is unknown (M_UNKNOWN_TOKEN error).
// This value is never returned. It's just used internally for caching.
const userIdUnknownToken = "."

type UserMappingResolver struct {
	logger                *logrus.Logger
	homeserverApiEndpoint string

	accessTokenToUserIdCacheMap *lru.TwoQueueCache
}

func NewUserMappingResolver(
	logger *logrus.Logger,
	cache *lru.TwoQueueCache,
	homeserverApiEndpoint string,
) *UserMappingResolver {
	return &UserMappingResolver{
		logger:                      logger,
		homeserverApiEndpoint:       homeserverApiEndpoint,
		accessTokenToUserIdCacheMap: cache,
	}
}

func (me *UserMappingResolver) ResolveByAccessToken(accessToken string) (string, error) {
	me.logger.Infof("Resolve request for %s", accessToken)

	userId, exists := me.accessTokenToUserIdCacheMap.Get(accessToken)
	if exists {
		if userId == userIdUnknownToken {
			me.logger.Debugf("Unknown token, from cache")
			return "", fmt.Errorf("Unknown token (cached)")
		}

		me.logger.Debugf("Resolved to %s from cache", userId)
		return userId.(string), nil
	}

	me.logger.Infof("Need to contact server..")

	var resp ApiWhoAmIResponse
	matrixClient, _ := gomatrix.NewClient(me.homeserverApiEndpoint, "unknown user id", accessToken)
	err := matrixClient.MakeRequest("GET", matrixClient.BuildURL("/account/whoami"), nil, &resp)
	if err != nil {
		//Certain common and expected errors (M_UNKNOWN_TOKEN), we try to interpret and possibly cache.
		//Others, we just return blindly, without caching.
		if IsErrorWithCode(err, ErrorUnknownToken) {
			go me.accessTokenToUserIdCacheMap.Add(accessToken, userIdUnknownToken)
			return "", fmt.Errorf("Unknown token")
		}

		return "", err
	}

	go me.accessTokenToUserIdCacheMap.Add(accessToken, resp.UserId)

	me.logger.Debugf("Resolved to %s from server", resp.UserId)

	return resp.UserId, nil
}
