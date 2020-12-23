package httpgateway

import (
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"

	"github.com/sirupsen/logrus"
)

type HookRunner struct {
	policyStore *policy.Store
	executor    *hook.Executor
}

func NewHookRunner(policyStore *policy.Store, executor *hook.Executor) *HookRunner {
	return &HookRunner{
		policyStore: policyStore,
		executor:    executor,
	}
}

func (me *HookRunner) RunFirstMatchingType(eventType string, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) hook.ExecutionResult {
	policyObj := me.policyStore.Get()
	if policyObj == nil {
		logger.Warnf("Hook Runner: service unavailable (missing policy)")

		httphelp.RespondWithMatrixError(
			w,
			http.StatusServiceUnavailable,
			matrix.ErrorUnknown,
			"Policy does not exist (yet), cannot proceed",
		)

		return hook.ExecutionResult{
			ResponseSent: true,
		}
	}

	for _, hookObj := range policyObj.Hooks {
		if hookObj.EventType == eventType && hookObj.MatchesRequest(request) {
			logger = logger.WithField("hookId", hookObj.ID)
			return me.runHook(hookObj, w, request, logger)
		}
	}

	return hook.ExecutionResult{
		Hook: nil,
	}
}

func (me *HookRunner) runHook(hookObj *hook.Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) hook.ExecutionResult {
	logger.Debugf("Hook Runner: executing hook: %s\n", hookObj.String())

	result := me.executor.Execute(hookObj, w, request, logger)

	logger.Debugf("Hook Runner: result: %#v\n", result)

	if result.ProcessingError != nil {
		logger = logger.WithField("error", result.ProcessingError)

		logger.Errorf("Hook Runner: encountered processing error, so we're sending a response\n")

		httphelp.RespondWithMatrixError(
			w,
			http.StatusServiceUnavailable,
			matrix.ErrorUnknown,
			"Hook execution failed, cannot proceed",
		)

		result.ResponseSent = true
	}

	return result
}
