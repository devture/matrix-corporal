package hookrunner

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

func (me *HookRunner) RunAllMatchingType(eventType string, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) hook.ExecutionResult {
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

	executedHooks := make([]*hook.Hook, 0)
	httpResponseModifierFuncs := make([]hook.HttpResponseModifierFunc, 0)

	logger = logger.WithField("hookEventType", eventType)

	for _, hookObj := range policyObj.Hooks {
		if hookObj.EventType != eventType || !hookObj.MatchesRequest(request) {
			continue
		}

		executedHooks = append(executedHooks, hookObj)

		logger = logger.WithField("hookId", hookObj.ID)

		// The chain also includes the current hook
		logger = logger.WithField("hookChain", hook.ListToChain(executedHooks))

		executionResult := me.runHook(hookObj, w, request, logger)

		httpResponseModifierFuncs = append(httpResponseModifierFuncs, executionResult.ReverseProxyResponseModifiers...)

		if !executionResult.NextHooksInChainCanRun() {
			// This is the end of the road for this execution chain.
			// The last hook either sent a response, or hit an error, or explicitly requested
			// hook execution to be aborted.
			return hook.ExecutionResult{
				ResponseSent:                  executionResult.ResponseSent,
				ProcessingError:               executionResult.ProcessingError,
				Hooks:                         executedHooks,
				ReverseProxyResponseModifiers: httpResponseModifierFuncs,
			}
		}

		// We can safely proceed to the next hook.
	}

	// If we're here, we either did not run any hooks, or none of them encountered
	// a processing error or sent a response.
	return hook.ExecutionResult{
		ResponseSent:                  false,
		ProcessingError:               nil,
		Hooks:                         executedHooks,
		ReverseProxyResponseModifiers: httpResponseModifierFuncs,
	}
}

func (me *HookRunner) runHook(hookObj *hook.Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) hook.ExecutionResult {
	logger.Infof("Executing hook")

	result := me.executor.Execute(hookObj, w, request, logger)

	logger.Debugf("Hook execution result: %#v\n", result)

	if result.ProcessingError != nil {
		logger = logger.WithField("error", result.ProcessingError)

		logger.Errorf("Hook Runner: encountered processing error, so we're sending a response")

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
