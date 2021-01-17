package hook

type ExecutionResult struct {
	Hooks []*Hook

	// ResponseSent indicates that executing the hook made it write some response.
	// In such cases, we wish to prevent further execution like:
	// - reverse-proxying, when handling `before*` hooks
	// - or delivering the reverse-proxy response, when handling `after*` hooks
	ResponseSent bool

	// SkipNextHooksInChain specifies whether a hook's execution resulted in a request to skip the next hooks.
	//
	// Setting this to false does not necessarily mean that execution continues.
	// This depends on other factors as well. See `NextHooksInChainCanRun()`.
	SkipNextHooksInChain bool

	ProcessingError error

	ReverseProxyResponseModifiers []HttpResponseModifierFunc
}

func (me ExecutionResult) NextHooksInChainCanRun() bool {
	return !(me.SkipNextHooksInChain || me.ResponseSent || me.ProcessingError != nil)
}

func createProcessingErrorExecutionResult(hook *Hook, err error) ExecutionResult {
	return ExecutionResult{
		Hooks:                []*Hook{hook},
		ProcessingError:      err,
		SkipNextHooksInChain: true,
	}
}
