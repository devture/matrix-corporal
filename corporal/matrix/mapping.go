package matrix

import (
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/matrix-org/gomatrix"
	"github.com/sirupsen/logrus"
)

// userIdUnknownToken is a special mapping value for when the access token is unknown (M_UNKNOWN_TOKEN error).
// This value is never returned. It's just used internally for caching.
const userIdUnknownToken = "."

type accessTokenResolvingResult struct {
	matrixUserID       string
	expiresAtTimestamp int64
}

type UserMappingResolver struct {
	logger                      *logrus.Logger
	accessTokenToUserIdCacheMap *lru.TwoQueueCache
	homeserverApiEndpoint       string
	expirationTimeMilliseconds  int64
}

func NewUserMappingResolver(
	logger *logrus.Logger,
	homeserverApiEndpoint string,
	cache *lru.TwoQueueCache,
	expirationTimeMilliseconds int64,
) *UserMappingResolver {
	return &UserMappingResolver{
		logger:                      logger,
		homeserverApiEndpoint:       homeserverApiEndpoint,
		accessTokenToUserIdCacheMap: cache,
		expirationTimeMilliseconds:  expirationTimeMilliseconds,
	}
}

func (me *UserMappingResolver) ResolveByAccessToken(accessToken string) (string, error) {
	me.logger.Debugf("Resolve request for token %s", accessToken)

	cachedResultInterface, exists := me.accessTokenToUserIdCacheMap.Get(accessToken)
	if exists {
		cachedResult := cachedResultInterface.(accessTokenResolvingResult)

		if int64(cachedResult.expiresAtTimestamp) > time.Now().Unix() {
			if cachedResult.matrixUserID == userIdUnknownToken {
				me.logger.Debugf("Unknown token, from cache")
				return "", fmt.Errorf("Unknown token (cached)")
			}

			me.logger.Debugf("Resolved to %s from cache", cachedResult.matrixUserID)
			return cachedResult.matrixUserID, nil
		}

		me.logger.Debugf("Found stale result in resolver cache")
	}

	me.logger.Debugf("Need to contact server..")

	var resp ApiWhoAmIResponse
	matrixClient, _ := gomatrix.NewClient(me.homeserverApiEndpoint, "unknown user id", accessToken)
	err := matrixClient.MakeRequest("GET", matrixClient.BuildURL("/account/whoami"), nil, &resp)
	if err != nil {
		// Certain common and expected errors (M_UNKNOWN_TOKEN), we try to interpret and possibly cache.
		// Others, we just return blindly, without caching.
		if IsErrorWithCode(err, ErrorUnknownToken) {
			go me.accessTokenToUserIdCacheMap.Add(accessToken, accessTokenResolvingResult{
				matrixUserID:       userIdUnknownToken,
				expiresAtTimestamp: time.Now().Add(time.Duration(me.expirationTimeMilliseconds) * time.Millisecond).Unix(),
			})

			return "", fmt.Errorf("Unknown token")
		}

		return "", err
	}

	result := accessTokenResolvingResult{
		matrixUserID:       resp.UserId,
		expiresAtTimestamp: time.Now().Add(time.Duration(me.expirationTimeMilliseconds) * time.Millisecond).Unix(),
	}

	go me.accessTokenToUserIdCacheMap.Add(accessToken, result)

	me.logger.Debugf("Resolved access token to %s from server", resp.UserId)

	return resp.UserId, nil
}
