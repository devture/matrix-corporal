package provider

type Provider interface {
	Type() string

	// Start starts the policy provider.
	// The provider may validate its configuration and return an error immediately.
	Start() error

	// Stop stops the policy provider.
	Stop()

	// Reload makes sure that the policy provider fetches fresh data.
	// Providers may or may not use caching as a fallback for their normal operation,
	// but when explicitly asked to reload, they must avoid caching.
	Reload()
}
