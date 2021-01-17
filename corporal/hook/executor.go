package hook

import (
	"bytes"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type executionHandler func(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult

type Executor struct {
	restServiceConsultor *RESTServiceConsultor

	actionToHandlerMap map[string]executionHandler
}

func NewExecutor(restServiceConsultor *RESTServiceConsultor) *Executor {
	me := &Executor{
		restServiceConsultor: restServiceConsultor,
	}

	me.actionToHandlerMap = map[string]executionHandler{
		ActionConsultRESTServiceURL: me.executeActionConsultRESTServiceURL,
		ActionReject:                executeActionReject,
		ActionRespond:               executeActionRespond,
		ActionPassUnmodified:        executePassUnmodified,
		ActionPassModifiedRequest:   executePassModifiedRequest,
		ActionPassModifiedResponse:  executePassModifiedResponse,
	}

	return me
}

func (me *Executor) Execute(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
	handler, exists := me.actionToHandlerMap[hookObj.Action]
	if !exists {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Missing handler for hook action = %s", hookObj.Action))
	}

	if strings.HasPrefix(hookObj.EventType, "before") {
		return me.executeBeforeHook(handler, hookObj, w, request, logger)
	}

	if strings.HasPrefix(hookObj.EventType, "after") {
		return me.executeAfterHook(handler, hookObj, w, request, logger)
	}

	return me.executeTypelessHook(handler, hookObj, w, request, logger)
}

// executeBeforeHook executes a hook of type `before*`.
//
// These hooks execute immediately.
// Depending on the hook's Action, they may return an HTTP response modifier function or not.
func (me *Executor) executeBeforeHook(
	handler executionHandler,
	hookObj *Hook,
	w http.ResponseWriter,
	request *http.Request,
	logger *logrus.Entry,
) ExecutionResult {
	return handler(hookObj, w, request, nil /* response */, logger)
}

// executeTypelessHook executes a hook which has no type.
//
// These hooks are the result of consulting a REST service.
//
// Hooks coming from REST services are not really a "before" or "after" hook.
// They're just hooks that get executed immediately.
func (me *Executor) executeTypelessHook(
	handler executionHandler,
	hookObj *Hook,
	w http.ResponseWriter,
	request *http.Request,
	logger *logrus.Entry,
) ExecutionResult {
	return handler(hookObj, w, request, nil /* response */, logger)
}

// executeAfterHook "executes" a hook of type `after*`.
//
// These hooks are meant to run after a response comes from the upstream.
//
// We say "execute", because after-hooks are merely scheduled to execute,
// by wrapping the normal hook logic into an HTTP response modifier function (regardless of the Action type).
//
// Nothing executes right now. We merely prepare stuff
// and wait for our HTTP response modifier function to get called.
func (me *Executor) executeAfterHook(
	handler executionHandler,
	hookObj *Hook,
	w http.ResponseWriter,
	request *http.Request,
	logger *logrus.Entry,
) ExecutionResult {
	// Some `after*`` hooks (like ActionConsultRESTServiceURL) need to read the request body.
	//
	// After-hooks run from a "reverse-proxy HTTP response modifier" function.
	// At the time this function executes, we would have already forwarded the original request
	// to the upstream via the reverse-proxy. This exhausts the request body reader.
	//
	// Unless we preserve it, we won't be able to use it for our hook's purposes.
	//
	// We don't capture/restore it for each type of after-hook action, because it's wasteful.
	// We only capture it for the action types we know will need it.
	var requestBodyBytes []byte

	if hookObj.Action == ActionConsultRESTServiceURL {
		var err error

		requestBodyBytes, err = httphelp.GetRequestBody(request)
		if err != nil {
			return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Could not get request body needed for %s after hook", hookObj.Action))
		}
	}

	if len(requestBodyBytes) > 0 {
		logger.Debugf(
			"Preserved %d bytes of request payload so the hook can use it\n",
			len(requestBodyBytes),
		)
	}

	var responseModifier HttpResponseModifierFunc = func(response *http.Response) ( /* skipNextModifiers */ bool, error) {
		logger.Debugln("In after-hook response modifier")

		if len(requestBodyBytes) > 0 {
			request.Body = ioutil.NopCloser(bytes.NewReader(requestBodyBytes))

			logger.Debugf(
				"Restored %d bytes of request payload so the hook can use it\n",
				len(requestBodyBytes),
			)
		}

		// Instead of passing the writer for the original response (`w`) to the handler,
		// we'd like to pass another one.
		//
		// This forwarding writer will capture calls to `.Write(..)` and `WriteHeader(statusCode int)`,
		// and put that data into the `response` object.
		//
		// `after*` hooks run from the reverse-proxy's HTTP response modifier.
		// Given that we're this far, the reverse-proxy service will be keen on writing the response's headers and data it already has.
		//
		// If we (or rather, the handler we call below) starts writing on its own, we'll be writing multiple times -
		// first from the handler itself, then when we exit this "HTTP respone modifier" function and the reverse-proxy
		// finally decides to write the response it sees in response.Body (unless we unset it).
		//
		// This unsetting thing works, but calls for writing headers (especially the response status code) don't,
		// and we may get to call it twice, which leads to strange errors we'd rather not have.
		//
		// So, we do things in a cleaner way. We pass a "collect stuff and modify the original response" writer
		// and let our hook action handlers run their normal course. All their attempts to send a header or data
		// will be gathered and dumped into the `response`.
		// We can then let the reverse-proxy send it (as it does by default) and we're done.
		responseBoundWriter := httphelp.NewResponseBoundHttpWriter(response)
		defer responseBoundWriter.Commit()

		// We won't need to care about this execution result's `ResponseSent` field,
		// because due to `responseBoundWriter` we never really send out a response,
		// but rather just write it out into the `response` object.
		result := handler(hookObj, responseBoundWriter, request, response, logger)

		logger.Debugf("After-hook execution result: %#v\n", result)

		if result.ProcessingError != nil {
			logger = logger.WithField("error", result.ProcessingError)

			logger.Errorf("After-hook HTTP modifier response: error\n")

			// This gets sent to responseBoundWriter, so it ends up in the `response` object.
			// It doesn't really get written out just yet.
			httphelp.RespondWithMatrixError(
				responseBoundWriter,
				http.StatusServiceUnavailable,
				matrix.ErrorUnknown,
				"Afer-hook execution failed, cannot proceed",
			)

			return true, nil
		}

		if len(result.ReverseProxyResponseModifiers) != 0 {
			for _, modifier := range result.ReverseProxyResponseModifiers {
				logger.Debugln("Passing control to the action's response modifier")
				skipNextModifiers, err := modifier(response)
				logger.Debugln("Returned to the after-hook action's response modifier")

				if err != nil {
					return true, err
				}

				if skipNextModifiers {
					// This embedded response modifier (spawned from the execution of the hook)
					// asked that no one modifiers run.
					// We should both "break" here and also prevent other top-level response modifiers from running.
					return true, nil
				}
			}

			// All response modifiers ran successfully
			return false, nil
		}

		if result.SkipNextHooksInChain {
			logger.Debugf("After-hook execution result requested that we skip execution of all other hooks in the chain: %#v\n", result)
			return true, nil
		}

		// Ignoring `ResponseSent` here.

		logger.Debugln("Finished after-hook response modifier")

		return false, nil
	}

	return ExecutionResult{
		ReverseProxyResponseModifiers: []HttpResponseModifierFunc{responseModifier},
	}
}

func (me *Executor) executeActionConsultRESTServiceURL(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	// The result of consulting is another "hook".
	// It could specify Action = reject or something similar, including calling another REST service.

	// RESTServiceConsultor should also be doing this sanity check, but let's do it here as well, for consistency.
	if hookObj.RESTServiceURL == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("A RESTServiceURL is required"))
	}

	newHookObj, err := me.restServiceConsultor.Consult(request, response, *hookObj, logger)
	if err != nil {
		return createProcessingErrorExecutionResult(hookObj, err)
	}

	if newHookObj.ID == "" {
		newHookObj.ID = fmt.Sprintf("%s-unnamed-response", hookObj.ID)
	}

	if newHookObj.EventType != "" {
		// Hooks received from REST services should not contain an event type.
		// We call then typeless hooks, because they run immediately.
		//
		// We unset this because people returning `after*` hooks would confuse the flow:
		// - if an `after*` hook is returned from a `before*` hook's REST handler,
		//   then people might expect that hook to be scheduled for execution after the original request.
		//   This is misleading. While they may inject hooks that modify the response, they can't, for example,
		//   schedule a REST after hook dynamically.
		// - if an `after*` hook is returned from an `after*` hook and we call `Execute()` below, we end up
		//   in `executeAfterHook` again, which merely returns a new HTTP response modifier.
		//   Those are not meant to be called recursively, as they'll get confused with response body copying.
		logger.Warnf(
			"Switching REST service result hook (%s) from eventType = `%s` to typeless",
			newHookObj,
			newHookObj.EventType,
		)

		newHookObj.EventType = ""
	}

	if hookObj.IsAfterHook() && newHookObj.Action == ActionPassModifiedRequest {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf(
			"An after hook (%s) yielded a request-modification hook: %s. It makes no sense - it's already too late to modify the request",
			hookObj,
			newHookObj,
		))
	}

	exportedHookJSON, err := json.Marshal(newHookObj)
	if err != nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Failed exporting hook: %s", err))
	}

	// It's important to be able to debug these network-related hook results easily,
	// so we're dumping them into the debug log in detail.
	logger.Debugf("Hook Executor: %s provided a new hook response %s", *hookObj.RESTServiceURL, string(exportedHookJSON))

	executionResult := me.Execute(newHookObj, w, request, logger)
	executionResult.Hooks = []*Hook{hookObj}

	return executionResult
}

func executeActionReject(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	if hookObj.RejectionErrorCode == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("A rejection error code is required"))
	}
	if hookObj.RejectionErrorMessage == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("A rejection error message is required"))
	}

	responseStatusCode := http.StatusForbidden
	if hookObj.ResponseStatusCode != nil {
		responseStatusCode = *hookObj.ResponseStatusCode
	}

	httphelp.RespondWithMatrixError(
		w,
		responseStatusCode,
		*hookObj.RejectionErrorCode,
		*hookObj.RejectionErrorMessage,
	)

	return ExecutionResult{
		Hooks:        []*Hook{hookObj},
		ResponseSent: true,
		// Regardless of this SkipNextHooksInChain value, hook execution can't continue anyway.
		SkipNextHooksInChain: hookObj.SkipNextHooksInChain,
	}
}

func executeActionRespond(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	if hookObj.ResponseStatusCode == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("A response status code is required"))
	}

	contentType := "application/json"
	if hookObj.ResponseContentType != nil {
		contentType = *hookObj.ResponseContentType
	}

	var payloadBytes []byte

	if contentType != "application/json" || hookObj.ResponseSkipPayloadJSONSerialization {
		payloadString, ok := hookObj.ResponsePayload.(string)
		if !ok {
			return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Could not interpret payload as string"))
		}

		payloadBytes = []byte(payloadString)
	} else {
		// JSON payload and its serialization is expected

		serialized, err := json.Marshal(hookObj.ResponsePayload)
		if err != nil {
			return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Could not JSON-serialize payload"))
		}

		payloadBytes = serialized

	}

	httphelp.RespondWithBytes(
		w,
		*hookObj.ResponseStatusCode,
		contentType,
		payloadBytes,
	)

	return ExecutionResult{
		Hooks:        []*Hook{hookObj},
		ResponseSent: true,
		// Regardless of this SkipNextHooksInChain value, hook execution can't continue anyway.
		SkipNextHooksInChain: hookObj.SkipNextHooksInChain,
	}
}

func executePassUnmodified(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	return ExecutionResult{
		Hooks:                []*Hook{hookObj},
		ResponseSent:         false,
		SkipNextHooksInChain: hookObj.SkipNextHooksInChain,
	}
}

func executePassModifiedRequest(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	if hookObj.InjectJSONIntoRequest == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("injectJSONIntoRequest information is required"))
	}

	if (len(*hookObj.InjectJSONIntoRequest) == 0) &&
		(hookObj.InjectHeadersIntoResponse == nil || len(*hookObj.InjectHeadersIntoResponse) == 0) {
		// Optimization. If there's nothing to inject, we can skip modifying the response.
		return executePassUnmodified(hookObj, w, request, response, logger)
	}

	var requestPayload map[string]interface{}
	err := httphelp.GetJsonFromRequestBody(request, &requestPayload)
	if err != nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Failed to interpret original response body as JSON: %s", err))
	}

	for k, v := range *hookObj.InjectJSONIntoRequest {
		requestPayload[k] = v
	}

	newRequestBytes, err := json.Marshal(requestPayload)
	if err != nil {
		// We don't expect this to happen, but..
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Failed to serialize modified response payload as JSON: %s", err))
	}

	request.Body = ioutil.NopCloser(bytes.NewReader(newRequestBytes))
	request.ContentLength = int64(len(newRequestBytes))

	if hookObj.InjectHeadersIntoRequest != nil {
		for k, v := range *hookObj.InjectHeadersIntoRequest {
			request.Header.Set(k, v)
		}
	}

	return ExecutionResult{
		Hooks:                []*Hook{hookObj},
		SkipNextHooksInChain: hookObj.SkipNextHooksInChain,
	}
}

func executePassModifiedResponse(hookObj *Hook, w http.ResponseWriter, request *http.Request, response *http.Response, logger *logrus.Entry) ExecutionResult {
	if hookObj.InjectJSONIntoResponse == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("injectJSONIntoResponse information is required"))
	}

	if (len(*hookObj.InjectJSONIntoResponse) == 0) &&
		(hookObj.InjectHeadersIntoResponse == nil || len(*hookObj.InjectHeadersIntoResponse) == 0) {
		// Optimization. If there's nothing to inject, we can skip modifying the response.
		return executePassUnmodified(hookObj, w, request, response, logger)
	}

	var responseModifier HttpResponseModifierFunc = func(response *http.Response) ( /* skipNextModifiers */ bool, error) {
		// We're operating under the assumption that the response contains a key-value JSON payload.
		// If this is not the case for some responses, we'll fail below.
		// This assumption and failure mode can be adjusted in the future, if necessary.

		var responsePayload map[string]interface{}
		err := httphelp.GetJsonFromResponseBody(response, &responsePayload)
		if err != nil {
			logger.Errorf("Failed to interpret original response body as JSON: %s", err)

			// Returning the error to the HTTP reverse proxy will make this become a "bad gateway" response.
			// We're making the design decision to fail like that, instead of silently respond with the correct data,
			// and somewhat "swallowing" this errors (even though we've logged it already).
			// We'd better fail hard when there's an expectation mismatch.
			// We may make this behavior customizable in the future, if necessary.
			return true, err
		}

		for k, v := range *hookObj.InjectJSONIntoResponse {
			responsePayload[k] = v
		}

		newResponseBytes, err := json.Marshal(responsePayload)
		if err != nil {
			// We don't expect this to happen, but..
			logger.Errorf("Failed to serialize modified response payload as JSON: %s", err)

			return true, err
		}

		response.Body = ioutil.NopCloser(bytes.NewReader(newResponseBytes))
		response.ContentLength = int64(len(newResponseBytes))

		if hookObj.InjectHeadersIntoResponse != nil {
			for k, v := range *hookObj.InjectHeadersIntoResponse {
				response.Header.Set(k, v)
			}
		}

		return false, nil
	}

	return ExecutionResult{
		Hooks:                         []*Hook{hookObj},
		ReverseProxyResponseModifiers: []HttpResponseModifierFunc{responseModifier},
		SkipNextHooksInChain:          hookObj.SkipNextHooksInChain,
	}
}
