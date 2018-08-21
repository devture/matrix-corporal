package userauth

import (
	"crypto/sha256"
	"fmt"

	"github.com/sirupsen/logrus"

	lru "github.com/hashicorp/golang-lru"
)

// CacheFallbackAuthenticator is a user authenticator which wraps another authenticator for resilience.
//
// The parent authenticator is always attempted first, as our aim is to get reliable data each and every time.
// In case it fails, however, we'll fallback to our locally cached data.
//
// This is especially useful for the RestAuthenticator.
// In case the remote REST server is down, we'd still like to continue serving the last-seen
// authentication data, at least for a while (until the data gets evicted from the LRU cache).
type CacheFallbackAuthenticator struct {
	authType string
	other    Authenticator
	cache    *lru.Cache
	logger   *logrus.Logger
}

func NewCacheFallackAuthenticator(
	authType string,
	other Authenticator,
	cache *lru.Cache,
	logger *logrus.Logger,
) *CacheFallbackAuthenticator {
	return &CacheFallbackAuthenticator{
		authType: authType,
		other:    other,
		cache:    cache,
		logger:   logger,
	}
}

func (me *CacheFallbackAuthenticator) Type() string {
	return me.authType
}

func (me *CacheFallbackAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	cacheKeyRaw := fmt.Sprintf("%s-%s-%s", userId, givenPassword, authCredential)
	m := sha256.New()
	m.Write([]byte(cacheKeyRaw))
	cacheKey := fmt.Sprintf("%s", m.Sum(nil))

	isAuthenticated, errUpstream := me.other.Authenticate(userId, givenPassword, authCredential)

	if errUpstream == nil {
		// Save this result for later, in case of upstream authenticator failure.
		me.cache.Add(cacheKey, isAuthenticated)

		return isAuthenticated, nil
	}

	// Upstream authenticator failed. See if we can fall back to our cache.
	cacheResult, ok := me.cache.Get(cacheKey)
	if !ok {
		return false, fmt.Errorf("Cache fallback failed, after upstream's failure: %s", errUpstream)
	}

	me.logger.Infof(
		"Retrieved cached auth result for user %s after upstream authenticator failed: %s",
		userId,
		errUpstream,
	)

	return cacheResult.(bool), nil
}
