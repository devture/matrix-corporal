package provider

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/policy"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// LastSeenStorePolicyProvider is a policy provider which restores the last-seen policy in the store.
// It also listens for new policies arriving to the store and saves them to a local file
// (for future restoration, if necessary).
//
// This is meant to be used when wishing to provide policies via push.
// That is, policies will come from an external system through the HTTP API (see the httpapi package).
// On service restart, however, until a new push arrives to the API, we want to restore the last-seen policy,
// which is what this policy provider does.
type LastSeenStorePolicyProvider struct {
	store     *policy.Store
	cachePath string
	logger    *logrus.Logger

	lockSave sync.Mutex
	channel  chan *policy.Policy
}

func NewLastSeenStorePolicyProvider(
	config configuration.PolicyProvider,
	store *policy.Store,
	logger *logrus.Logger,
) (*LastSeenStorePolicyProvider, error) {
	cachePath, exists := config["CachePath"]
	if !exists {
		return nil, fmt.Errorf("Last Seen Store Policy provider requires a CachePath")
	}

	return &LastSeenStorePolicyProvider{
		store:     store,
		cachePath: cachePath.(string),
		logger:    logger,
	}, nil
}

func (me *LastSeenStorePolicyProvider) Type() string {
	return "last_seen_store_policy"
}

func (me *LastSeenStorePolicyProvider) Start() error {
	me.logger.Infof("Starting policy provider: %s", me.Type())

	err := me.load()

	if err != nil {
		return err
	}

	me.channel = me.store.GetNotificationChannel()
	go me.listenOnChannel(me.channel)

	return nil
}

func (me *LastSeenStorePolicyProvider) Stop() {
	me.logger.Infof("Stopping policy provider: %s", me.Type())

	if me.channel != nil {
		me.store.DestroyNotificationChannel(me.channel)
	}
}

func (me *LastSeenStorePolicyProvider) Reload() {
	me.logger.Infof("Ignoring Reload command in policy provider: %s", me.Type())
}

func (me *LastSeenStorePolicyProvider) load() error {
	file, err := os.Open(me.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If we don't have anything cached, that's OK.
			return nil
		}

		return err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	policy, err := createPolicyFromJsonBytes(bytes)
	if err != nil {
		return fmt.Errorf("Policy load error: %s", err)
	}

	err = me.store.Set(policy)
	if err != nil {
		return fmt.Errorf("Policy set error: %s", err)
	}

	return nil
}

func (me *LastSeenStorePolicyProvider) listenOnChannel(channel chan *policy.Policy) {
	for {
		policy, more := <-channel

		if !more {
			return
		}

		me.lockSave.Lock()

		me.logger.Infof("Saving new policy that arrived at the policy store")
		err := me.storePolicyInCache(policy)
		if err == nil {
			me.logger.Infof("Policy saving completed")
		} else {
			me.logger.Warnf("Policy saving failed: %s", err)
		}

		me.lockSave.Unlock()
	}
}

func (me *LastSeenStorePolicyProvider) storePolicyInCache(policy *policy.Policy) error {
	jsonBytes, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	file, err := os.Create(me.cachePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonBytes)
	if err != nil {
		return err
	}

	return nil
}
