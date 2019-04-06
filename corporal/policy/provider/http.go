package provider

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/policy"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type HttpProvider struct {
	store                    *policy.Store
	uri                      string
	authorizationBearerToken string
	cachePath                *string
	reloadIntervalSeconds    *int
	logger                   *logrus.Logger

	httpClient   *http.Client
	reloadTicker *time.Ticker
	lockLoad     sync.Mutex
}

func NewHttpProvider(
	config configuration.PolicyProvider,
	store *policy.Store,
	logger *logrus.Logger,
) (*HttpProvider, error) {
	configKeys := []string{
		"Uri",
		"AuthorizationBearerToken",
		"CachePath",
		"ReloadIntervalSeconds",
		"TimeoutMilliseconds",
	}

	for _, key := range configKeys {
		_, ok := config[key]
		if !ok {
			return nil, fmt.Errorf("HTTP provider is missing a required configuration key: %s", key)
		}
	}

	var cachePathPtr *string
	if config["CachePath"] != nil {
		cachePath := config["CachePath"].(string)
		cachePathPtr = &cachePath
	}

	var reloadIntervalSecondsPtr *int
	if config["ReloadIntervalSeconds"] != nil {
		reloadIntervalSecondsFloat, ok := config["ReloadIntervalSeconds"].(float64)
		if !ok {
			return nil, fmt.Errorf("ReloadIntervalSeconds is expected to be a number or NULL")
		}
		reloadIntervalSeconds := int(reloadIntervalSecondsFloat)
		if reloadIntervalSeconds > 0 {
			reloadIntervalSecondsPtr = &reloadIntervalSeconds
		}
	}

	var timeoutDuration time.Duration
	if config["TimeoutMilliseconds"] != nil {
		timeoutMillisecondsFloat, ok := config["TimeoutMilliseconds"].(float64)
		if !ok {
			return nil, fmt.Errorf("TimeoutMilliseconds is expected to be a number or NULL")
		}
		if timeoutMillisecondsFloat > 0 {
			timeoutDuration = time.Duration(timeoutMillisecondsFloat) * time.Millisecond
		}
	}

	return &HttpProvider{
		store: store,
		uri:   config["Uri"].(string),
		authorizationBearerToken: config["AuthorizationBearerToken"].(string),
		cachePath:                cachePathPtr,
		reloadIntervalSeconds:    reloadIntervalSecondsPtr,
		logger:                   logger,

		httpClient: &http.Client{
			Timeout: timeoutDuration,
		},
	}, nil
}

func (me *HttpProvider) Type() string {
	return "http"
}

func (me *HttpProvider) Start() error {
	me.logger.Infof("Starting policy provider: %s (%s)", me.Type(), me.uri)

	err := me.load(true)

	if err != nil {
		return err
	}

	if me.reloadIntervalSeconds != nil {
		me.logger.Infof("Auto-reloading for policy provider %s will happen every %d seconds", me.Type(), *me.reloadIntervalSeconds)

		me.reloadTicker = time.NewTicker(time.Duration(*me.reloadIntervalSeconds) * time.Second)

		go func() {
			for range me.reloadTicker.C {
				me.logger.Infof("Auto-reloading for policy provider: %s", me.Type())
				me.Reload()
			}
		}()
	}

	return nil
}

func (me *HttpProvider) Stop() {
	me.logger.Infof("Stopping policy provider: %s", me.Type())

	if me.reloadTicker != nil {
		me.reloadTicker.Stop()
	}
}

func (me *HttpProvider) Reload() {
	me.logger.Infof("Reloading policy from provider: %s", me.Type())

	err := me.load(false)
	if err != nil {
		me.logger.Infof("Failed reloading policy: %s", err)
	}
}

func (me *HttpProvider) load(allowedToLoadFromCache bool) error {
	me.lockLoad.Lock()
	defer me.lockLoad.Unlock()

	policy, isFromCache, err := me.doLoad(allowedToLoadFromCache)
	if err != nil {
		return err
	}

	if !isFromCache {
		me.storePolicyInCache(policy)
	}

	err = me.store.Set(policy)
	if err != nil {
		return fmt.Errorf("Policy set error: %s", err)
	}

	return nil
}

func (me *HttpProvider) doLoad(allowedToLoadFromCache bool) (*policy.Policy /* isFromCache */, bool, error) {
	policy, errRemote := me.loadPolicyFromRemote()
	if errRemote == nil {
		me.logger.Debugf("Successfully loaded policy from URL: %s", me.uri)
		return policy, false, nil
	}

	me.logger.Warnf("Failed loading policy from URL (%s): %s", me.uri, errRemote)

	if !allowedToLoadFromCache {
		return nil, false, fmt.Errorf("Failed loading policy from remote (%s), while cache-loading is not allowed", errRemote)
	}

	policy, errCache := me.loadPolicyFromCache()
	if errCache == nil {
		me.logger.Debugf("Successfully loaded policy from cache")
		return policy, true, nil
	}

	return nil, false, fmt.Errorf("Failed loading policy from remote (%s) and from cache (%s)", errRemote, errCache)
}

func (me *HttpProvider) loadPolicyFromRemote() (*policy.Policy, error) {
	req, err := http.NewRequest("GET", me.uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", me.authorizationBearerToken))

	resp, err := me.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 response fetching from URL: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed reading HTTP response body: %s", err)
	}

	return createPolicyFromJsonBytes(bodyBytes)
}

func (me *HttpProvider) loadPolicyFromCache() (*policy.Policy, error) {
	if me.cachePath == nil {
		return nil, fmt.Errorf("Cache disabled")
	}

	file, err := os.Open(*me.cachePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return createPolicyFromJsonBytes(bytes)
}

func (me *HttpProvider) storePolicyInCache(policy *policy.Policy) error {
	if me.cachePath == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	file, err := os.Create(*me.cachePath)
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
