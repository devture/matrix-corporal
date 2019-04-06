package provider

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/policy"
	"fmt"

	"github.com/sirupsen/logrus"
)

func CreateProviderByConfig(
	config configuration.PolicyProvider,
	store *policy.Store,
	logger *logrus.Logger,
) (Provider, error) {
	providerType, exists := config["Type"]
	if !exists {
		return nil, fmt.Errorf("Provider configuration is missing a type: %#v", config)
	}

	if providerType == "static_file" {
		return NewStaticFileProvider(config, store, logger)
	}

	if providerType == "http" {
		return NewHttpProvider(config, store, logger)
	}

	if providerType == "last_seen_store_policy" {
		return NewLastSeenStorePolicyProvider(config, store, logger)
	}

	return nil, fmt.Errorf("Unknown provider type: %s", providerType)
}
