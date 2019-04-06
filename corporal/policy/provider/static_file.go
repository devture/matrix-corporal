package provider

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/policy"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

type StaticFileProvider struct {
	store  *policy.Store
	path   string
	logger *logrus.Logger

	lockLoad sync.Mutex
	watcher  *fsnotify.Watcher
}

func NewStaticFileProvider(
	config configuration.PolicyProvider,
	store *policy.Store,
	logger *logrus.Logger,
) (*StaticFileProvider, error) {
	path, exists := config["Path"]
	if !exists {
		return nil, fmt.Errorf("Static file provider requires a Path")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("Failed initializing inotify watcher: %s", err)
	}

	return &StaticFileProvider{
		store:  store,
		path:   path.(string),
		logger: logger,

		watcher: watcher,
	}, nil
}

func (me *StaticFileProvider) Type() string {
	return "static_file"
}

func (me *StaticFileProvider) Start() error {
	me.logger.Infof("Starting policy provider: %s", me.Type())

	err := me.load()

	if err != nil {
		return err
	}

	go me.watch()

	return nil
}

func (me *StaticFileProvider) Stop() {
	me.logger.Infof("Stopping policy provider: %s", me.Type())

	me.watcher.Close()
}

func (me *StaticFileProvider) Reload() {
	me.logger.Infof("Reloading policy from provider: %s", me.Type())

	err := me.load()

	if err != nil {
		me.logger.Infof("Failed reloading policy: %s", err)
	}
}

func (me *StaticFileProvider) load() error {
	me.lockLoad.Lock()
	defer me.lockLoad.Unlock()

	file, err := os.Open(me.path)
	if err != nil {
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

func (me *StaticFileProvider) watch() {
	go func() {
		for {
			select {
			case ev := <-me.watcher.Events:
				// We handle remove events too, because editors like vim would swap the file atomically.
				// There's no Write operation there, rather a sequence (Rename, Chmod, Remove)
				// (could be a bug too, see: https://github.com/fsnotify/fsnotify/issues/92)
				isWrite := ev.Op&fsnotify.Write == fsnotify.Write
				isRemove := ev.Op&fsnotify.Remove == fsnotify.Remove

				if !isWrite && !isRemove {
					continue
				}

				time.AfterFunc(time.Duration(1*time.Second), func() {
					err := me.load()

					if err == nil {
						me.logger.Infof("Reloaded policy from %s", me.path)
					} else {
						me.logger.Warnf("Failed to reload policy from %s: %s", me.path, err)
					}
				})

				// If the file gets removed, we need to start watching it again.
				if isRemove {
					me.watcher.Add(me.path)
				}
			}
		}
	}()

	me.watcher.Add(me.path)
}
