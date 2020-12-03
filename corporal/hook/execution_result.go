package hook

import "net/http"

type HttpResponseModifierFunc func(*http.Response) error

type ExecutionResult struct {
	Hook *Hook

	SkipProceedingFurther bool

	ProcessingError error

	ReverseProxyResponseModifier *HttpResponseModifierFunc
}

func createProcessingErrorExecutionResult(hook *Hook, err error) ExecutionResult {
	return ExecutionResult{
		Hook:                  hook,
		SkipProceedingFurther: true,
		ProcessingError:       err,
	}
}
