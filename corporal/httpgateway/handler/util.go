package handler

import (
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httpgateway/hookrunner"
	"net/http"

	"github.com/sirupsen/logrus"
)

// runHook runs the first matching hook of a given type, possibly injects a response modifier and returns false if we should stop execution
func runHook(
	hookRunner *hookrunner.HookRunner,
	eventType string,
	w http.ResponseWriter,
	r *http.Request,
	logger *logrus.Entry,
	httpResponseModifierFuncs *[]hook.HttpResponseModifierFunc,
) bool {
	hookResult := hookRunner.RunFirstMatchingType(eventType, w, r, logger)
	if hookResult.ResponseSent {
		logger.WithField("hookId", hookResult.Hook.ID).WithField("hookEventType", hookResult.Hook.EventType).Infoln(
			"HTTP gateway (policy-checked): hook delivered a response, so we're not proceeding further",
		)
		return false
	}

	if hookResult.ReverseProxyResponseModifier != nil {
		*httpResponseModifierFuncs = append(*httpResponseModifierFuncs, *hookResult.ReverseProxyResponseModifier)
	}

	return true
}
