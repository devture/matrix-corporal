package handler

import (
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httpgateway/hookrunner"
	"net/http"

	"github.com/sirupsen/logrus"
)

// runHooks runs all matching hook of a given type, possibly injects a response modifier and returns false if we should stop execution
func runHooks(
	hookRunner *hookrunner.HookRunner,
	eventType string,
	w http.ResponseWriter,
	r *http.Request,
	logger *logrus.Entry,
	httpResponseModifierFuncs *[]hook.HttpResponseModifierFunc,
) bool {
	hookResult := hookRunner.RunAllMatchingType(eventType, w, r, logger)
	if hookResult.ResponseSent {
		logger.WithField("hookChain", hook.ListToChain(hookResult.Hooks)).Infoln(
			"HTTP gateway (policy-checked): hook delivered a response, so we're not proceeding further",
		)
		return false
	}

	*httpResponseModifierFuncs = append(*httpResponseModifierFuncs, hookResult.ReverseProxyResponseModifiers...)

	return true
}
