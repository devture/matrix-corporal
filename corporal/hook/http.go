package hook

import (
	"net/http"
)

type HttpResponseModifierFunc func(*http.Response) ( /* skipNextModifiers */ bool, error)

// CreateChainedHttpResponseModifierFunc chains a list of HttpResponseModifierFunc functions into a single function.
//
// This is useful, because the reverse-proxy only takes a single `reverseProxy.ModifyResponse` modifier function,
// yet we sometimes wish to schedule multiple modifiers.
func CreateChainedHttpResponseModifierFunc(functions []HttpResponseModifierFunc) func(*http.Response) error {
	return func(response *http.Response) error {
		for _, function := range functions {
			skipNextModifiers, err := function(response)
			if err != nil {
				return err
			}

			if skipNextModifiers {
				// No error, but we've been asked to stop
				return nil
			}
		}
		return nil
	}
}
