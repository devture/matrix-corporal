package configuration

import (
	"devture-matrix-corporal/corporal/matrix"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type Configuration struct {
	Matrix         Matrix
	Corporal       Corporal
	Reconciliation Reconciliation
	HttpApi        HttpApi
	HttpGateway    HttpGateway
	PolicyProvider PolicyProvider
	Misc           Misc
}

type HttpApi struct {
	Enabled                  bool
	ListenAddress            string
	AuthorizationBearerToken string
	TimeoutMilliseconds      int
}

type HttpGateway struct {
	ListenAddress       string
	TimeoutMilliseconds int
	InternalRESTAuth    HttpGatewayInternalRESTAuth
	UserMappingResolver HttpGatewayUserMappingResolver
}

type HttpGatewayInternalRESTAuth struct {
	Enabled            *bool
	IPNetworkWhitelist *[]string
}

type HttpGatewayUserMappingResolver struct {
	// CacheSize specifies the number of access tokens to cache
	CacheSize int

	// ExpirationTimeMilliseconds specifies how long resolved user IDs are valid for.
	// After expiration, we'll re-resolve them.
	ExpirationTimeMilliseconds int64
}

type Matrix struct {
	HomeserverDomainName     string
	HomeserverApiEndpoint    string
	AuthSharedSecret         string
	RegistrationSharedSecret string
	ReconciliatorUserId      string
	TimeoutMilliseconds      int
}

type Corporal struct {
	UserID string
}

type Reconciliation struct {
	RetryIntervalMilliseconds int
}

type Misc struct {
	Debug bool
}

type PolicyProvider map[string]interface{}

func LoadConfiguration(filePath string, logger *logrus.Logger) (*Configuration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read configuration from %s: %s", filePath, err)
	}

	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode JSON: %s", err)
	}

	setConfigurationDefaults(&configuration)

	err = validateConfiguration(&configuration, logger)
	if err != nil {
		return nil, fmt.Errorf("Failed to validate configuration: %s", err)
	}

	return &configuration, nil
}

func setConfigurationDefaults(configuration *Configuration) {
	if configuration.HttpGateway.UserMappingResolver.CacheSize == 0 {
		configuration.HttpGateway.UserMappingResolver.CacheSize = 10000
	}

	if configuration.HttpGateway.UserMappingResolver.ExpirationTimeMilliseconds == 0 {
		configuration.HttpGateway.UserMappingResolver.ExpirationTimeMilliseconds = 5 * 60 * 1000
	}
}

func validateConfiguration(configuration *Configuration, logger *logrus.Logger) error {
	if !matrix.IsFullUserIdOfDomain(configuration.Corporal.UserID, configuration.Matrix.HomeserverDomainName) {
		return fmt.Errorf(
			"Reconciliation user `%s` (specified in Corporal.UserID) is not hosted on the managed homeserver domain (%s)",
			configuration.Corporal.UserID,
			configuration.Matrix.HomeserverDomainName,
		)
	}

	if configuration.Matrix.TimeoutMilliseconds <= 0 {
		return fmt.Errorf("Matrix.TimeoutMilliseconds needs to be a positive number")
	}

	if configuration.Reconciliation.RetryIntervalMilliseconds <= 0 {
		return fmt.Errorf("Reconciliation.RetryIntervalMilliseconds needs to be a positive number")
	}

	if configuration.HttpGateway.TimeoutMilliseconds <= 0 {
		return fmt.Errorf("HttpGateway.TimeoutMilliseconds needs to be a positive number")
	}
	if configuration.HttpGateway.TimeoutMilliseconds < configuration.Matrix.TimeoutMilliseconds {
		return fmt.Errorf(
			"HttpGateway.TimeoutMilliseconds (%d) needs to be larger than Matrix.TimeoutMilliseconds (%d)",
			configuration.HttpGateway.TimeoutMilliseconds,
			configuration.Matrix.TimeoutMilliseconds,
		)
	}
	if configuration.HttpGateway.InternalRESTAuth.Enabled == nil || (*configuration.HttpGateway.InternalRESTAuth.Enabled) == false {
		logger.Warn("HttpGateway.InternalRESTAuth.Enabled is neither explicitly enabled, nor disabled. Interactive Auth may not work without it. Define it as enabled or disabled to get rid of this warning")
	} else {
		if configuration.HttpGateway.InternalRESTAuth.IPNetworkWhitelist == nil {
			logger.Debug("No whitelisted IP ranges are defined in `HttpGateway.InternalRESTAuth.IPNetworkWhitelist`. Will default to local/private networks")
		} else {
			if len(*configuration.HttpGateway.InternalRESTAuth.IPNetworkWhitelist) == 0 {
				logger.Info("An empty IP range whitelist defined for HTTP Internal Auth. All IP addresses will be allowed")
			}
		}
	}

	if configuration.HttpApi.TimeoutMilliseconds <= 0 {
		return fmt.Errorf("HttpApi.TimeoutMilliseconds needs to be a positive number")
	}

	return nil
}
