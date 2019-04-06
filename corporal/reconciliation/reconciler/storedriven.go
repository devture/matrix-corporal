package reconciler

import (
	"devture-matrix-corporal/corporal/policy"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type StoreDrivenReconciler struct {
	logger                    *logrus.Logger
	store                     *policy.Store
	reconciler                *Reconciler
	retryIntervalMilliseconds int

	lockReconciler sync.Mutex
	channel        chan *policy.Policy
	retryTicker    *time.Ticker
	retryCancel    chan bool
}

func NewStoreDrivenReconciler(
	logger *logrus.Logger,
	store *policy.Store,
	reconciler *Reconciler,
	retryIntervalMilliseconds int,
) *StoreDrivenReconciler {
	return &StoreDrivenReconciler{
		logger:                    logger,
		store:                     store,
		reconciler:                reconciler,
		retryIntervalMilliseconds: retryIntervalMilliseconds,
	}
}

func (me *StoreDrivenReconciler) Start() error {
	me.channel = me.store.GetNotificationChannel()

	go me.listenOnChannel(me.channel)

	me.logger.Infof("Started store-driven reconciler")

	return nil
}

func (me *StoreDrivenReconciler) Stop() {
	me.store.DestroyNotificationChannel(me.channel)

	me.logger.Infof("Stopped store-driven reconciler")
}

func (me *StoreDrivenReconciler) listenOnChannel(channel chan *policy.Policy) {
	for {
		policy, more := <-channel

		if !more {
			return
		}

		me.logger.Infof("Store-driven reconciler received a new policy from the store")

		me.lockReconciler.Lock()

		// We may still be potentially retrying some old policy.
		// Let's stop that and attempt to load the new one below.
		if me.retryTicker != nil {
			me.retryTicker.Stop()
			me.retryCancel <- true

			me.retryTicker = nil
			me.retryCancel = nil
		}

		me.logger.Infof("Reconciling..")
		err := me.reconciler.Reconcile(policy)
		if err == nil {
			me.logger.Infof("Reconciliation completed")
		} else {
			me.logger.Warnf("Reconciliation failed: %s", err)
		}

		me.lockReconciler.Unlock()

		if err != nil {
			me.retryTicker = time.NewTicker(
				time.Duration(me.retryIntervalMilliseconds) * time.Millisecond,
			)
			// Buffered signalling channel, so we can avoid getting stuck if the retrier had exited
			me.retryCancel = make(chan bool, 1)
			go me.retryReconciliation(me.retryTicker, me.retryCancel, policy)
			me.logger.Infof("Will retry reconciliation after %d ms..", me.retryIntervalMilliseconds)
		}
	}
}

func (me *StoreDrivenReconciler) retryReconciliation(ticker *time.Ticker, cancel chan bool, policy *policy.Policy) {
	for {
		select {
		case <-ticker.C:
			me.lockReconciler.Lock()

			me.logger.Infof("Retrying reconciliation..")

			err := me.reconciler.Reconcile(policy)

			if err == nil {
				me.logger.Infof("Reconciliation completed")
				ticker.Stop()
				me.lockReconciler.Unlock()
				return
			}

			me.logger.Warnf("Reconciliation failed: %s", err)
			me.lockReconciler.Unlock()

		case <-cancel:
			return
		}
	}
}
