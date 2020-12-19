package hook

type ExecutionResult struct {
	Hook *Hook

	// ResponseSent indicates that executing the hook made it write some response.
	// In such cases, we wish to prevent further execution like:
	// - reverse-proxying, when handling `before*` hooks
	// - or delivering the reverse-proxy response, when handling `after*` hooks
	ResponseSent bool

	ProcessingError error

	ReverseProxyResponseModifier *HttpResponseModifierFunc
}

func createProcessingErrorExecutionResult(hook *Hook, err error) ExecutionResult {
	return ExecutionResult{
		Hook:            hook,
		ProcessingError: err,
	}
}
