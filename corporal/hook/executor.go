package hook

import (
	"bytes"
	"devture-matrix-corporal/corporal/httphelp"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

type executionHandler func(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult

type Executor struct {
	restServiceConsultor *RESTServiceConsultor

	actionToHandlerMap map[string]executionHandler
}

func NewExecutor(restServiceConsultor *RESTServiceConsultor) *Executor {
	me := &Executor{
		restServiceConsultor: restServiceConsultor,
	}

	me.actionToHandlerMap = map[string]executionHandler{
		ActionConsultRESTServiceURL:      me.executeActionConsultRESTServiceURL,
		ActionReject:                     executeActionReject,
		ActionRespond:                    executeActionRespond,
		ActionPassUnmodified:             executePassUnmodified,
		ActionPassInjectJSONIntoResponse: executePassInjectJSONIntoResponse,
	}

	return me
}

func (me *Executor) Execute(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
	handler, exists := me.actionToHandlerMap[hookObj.Action]
	if !exists {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Missing handler for hook action = %s", hookObj.Action))
	}

	return handler(hookObj, w, request, logger)
}

func (me *Executor) executeActionConsultRESTServiceURL(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
	// The result of consulting is another "hook".
	// It could specify Action = reject or something similar, including calling another REST service.

	// RESTServiceConsultor should also be doing this sanity check, but let's do it here as well, for consistency.
	if hookObj.RESTServiceURL == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("A RESTServiceURL is required"))
	}

	newHookObj, err := me.restServiceConsultor.Consult(request, *hookObj, logger)
	if err != nil {
		return ExecutionResult{
			Hook:                  hookObj,
			SkipProceedingFurther: true,
			ProcessingError:       err,
		}
	}

	if newHookObj.ID == "" {
		newHookObj.ID = fmt.Sprintf("%s-unnamed-response", hookObj.ID)
	}

	exportedHookJSON, err := json.Marshal(newHookObj)
	if err != nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("Failed exporting hook: %s", err))
	}

	// It's important to be able to debug these network-related hook results easily,
	// so we're dumping them into the debug log in detail.
	logger.Debugf("Hook Executor: %s provided a new hook response %s", *hookObj.RESTServiceURL, string(exportedHookJSON))

	response := me.Execute(newHookObj, w, request, logger)
	response.Hook = hookObj

	return response
}

func executeActionReject(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
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
		Hook:                  hookObj,
		SkipProceedingFurther: true,
	}
}

func executeActionRespond(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
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
		Hook:                  hookObj,
		SkipProceedingFurther: true,
	}
}

func executePassUnmodified(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
	return ExecutionResult{
		Hook:                  hookObj,
		SkipProceedingFurther: false,
	}
}

func executePassInjectJSONIntoResponse(hookObj *Hook, w http.ResponseWriter, request *http.Request, logger *logrus.Entry) ExecutionResult {
	if hookObj.InjectJSONIntoResponse == nil {
		return createProcessingErrorExecutionResult(hookObj, fmt.Errorf("injectJSONIntoResponse information is required"))
	}

	if (len(*hookObj.InjectJSONIntoResponse) == 0) &&
		(hookObj.InjectHeadersIntoResponse == nil || len(*hookObj.InjectHeadersIntoResponse) == 0) {
		// Optimization. If there's nothing to inject, we can skip modifying the response.
		return executePassUnmodified(hookObj, w, request, logger)
	}

	var responseModifier HttpResponseModifierFunc = func(response *http.Response) error {
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
			return err
		}

		for k, v := range *hookObj.InjectJSONIntoResponse {
			responsePayload[k] = v
		}

		newResponseBytes, err := json.Marshal(responsePayload)
		if err != nil {
			// We don't expect this to happen, but..

			logger.Errorf("Failed to serialize modified response payload as JSON: %s", err)

			return err
		}

		// We read the body, so we ought to restore it,
		// so that other things (like reverse-proxying) can read it later.
		response.Body = ioutil.NopCloser(bytes.NewReader(newResponseBytes))

		if hookObj.InjectHeadersIntoResponse != nil {
			for k, v := range *hookObj.InjectHeadersIntoResponse {
				response.Header.Set(k, v)
			}
		}

		return nil
	}

	return ExecutionResult{
		Hook:                  hookObj,
		SkipProceedingFurther: false,

		ReverseProxyResponseModifier: &responseModifier,
	}
}
