package hook

import "net/http"

type HttpResponseModifierFunc func(*http.Response) error

// CreateChainedHttpResponseModifierFunc chains a list of HttpResponseModifierFunc functions into a single function.
//
// This is useful, because the reverse-proxy only takes a single `reverseProxy.ModifyResponse` modifier function,
// yet we sometimes wish to schedule multiple modifiers.
func CreateChainedHttpResponseModifierFunc(functions []HttpResponseModifierFunc) HttpResponseModifierFunc {
	return func(response *http.Response) error {
		for _, function := range functions {
			err := function(response)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
