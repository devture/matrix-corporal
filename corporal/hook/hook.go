package hook

import (
	"devture-matrix-corporal/corporal/util"
	"fmt"
	"net/http"
	"strings"
)

// restActionHookDetails contains some fields which are useful when Hook.Action is something like ActionConsultRESTServiceURL
type restActionHookDetails struct {
	// RESTServiceURL specifies the URL of the REST service to call when Action = ActionConsultRESTServiceURL
	// Required field.
	RESTServiceURL *string `json:"RESTServiceURL,omitempty"`

	// RESTServiceRequestMethod specifies the request method to use when making the HTTP request RESTServiceURL
	// If not specified, a "POST" request will be used.
	RESTServiceRequestMethod *string `json:"RESTServiceRequestMethod,omitempty"`

	// RESTServiceRequestHeaders specifies any request headers that should be sent to the RESTServiceURL when making requests.
	//
	// Example:
	//	RESTServiceRequestHeaders = map[string]string{
	//		"Authorization": "Bearer: SOME_TOKEN",
	//	}
	RESTServiceRequestHeaders *map[string]string `json:"RESTServiceRequestHeaders,omitempty"`

	// RESTServiceRequestTimeoutMilliseconds specifies how long the HTTP request to RESTServiceURL is allowed to take.
	// If this is not defined, a default timeout value is used (30 seconds at the time of this writing).
	RESTServiceRequestTimeoutMilliseconds *uint `json:"RESTServiceRequestTimeoutMilliseconds,omitempty"`

	// RESTServiceRetryAttempts specifies how many times to retry the REST service HTTP request if failures are encountered.
	// If not specified, no retries will be attempted.
	RESTServiceRetryAttempts *uint `json:"RESTServiceRetryAttempts,omitempty"`

	// RESTServiceRetryWaitTimeMilliseconds specifies how long to wait between retries when contacting the REST service.
	// This only makes sense if RESTServiceRetryAttempts is set to a positive number.
	// If not specified, retries will happen immediately without waiting.
	RESTServiceRetryWaitTimeMilliseconds *uint `json:"RESTServiceRetryWaitTimeMilliseconds,omitempty"`

	// RESTServiceAsync specifies whether REST HTTP calls should be waited upon.
	// If not specified, we default to waiting on them and extracting their result (a new hook object).
	//
	// If this is set to true, we'll simply fire the request and not care about what the response is.
	// We'll still retry (obeying RESTServiceRetryAttempts and RESTServiceRetryWaitTimeMilliseconds) and expect an OK (200) response,
	// but it will no longer block the request, nor can it influence it.
	// The result of async REST hooks can be specified in RESTServiceAsyncResultHook.
	// By default (if not specified), we let the original request/response pass through unmodified.
	RESTServiceAsync bool `json:"RESTServiceAsync,omitempty"`

	// RESTServiceAsyncResultHook contains the hook to return as a result for RESTServiceAsync = true REST service calls.
	//
	// If not specified, RESTServiceAsync = true hooks's result is a new hook with Action = ActionPassUnmodified.
	RESTServiceAsyncResultHook *Hook `json:"RESTServiceAsyncResultHook,omitempty"`

	// RESTServiceContingencyHook contains a fallback hook to return as a result if the REST service fails.
	//
	// This can both be a communication failure or it returning a response we can't make sense of.
	//
	// If RESTServiceContingencyHook is not defined, any such REST service failures
	// cause execution to stop (503 / "service unavailable").
	RESTServiceContingencyHook *Hook `json:"RESTServiceContingencyHook,omitempty"`
}

type respondActionHookDetails struct {
	// Payload specifies the payload to respond with.
	// This may be some key-value JSON thing (`map[string]interface{}`), a string, etc.
	ResponsePayload interface{} `json:"responsePayload,omitempty"`

	// ResponseSkipPayloadJSONSerialization specifies whether the payload found in ResponsePayload should be JSON-serialized.
	// This only applies when ResponseContentType = "application/json".
	// This defaults to false. That is, we serialize to JSON by default (when ResponseContentType = "application/json").
	ResponseSkipPayloadJSONSerialization bool `json:"responseSkipPayloadJSONSerialization,omitempty"`

	// ResponseStatusCode specifies the HTTP response code that we'll be responding with.
	// Required field.
	ResponseStatusCode *int `json:"responseStatusCode,omitempty"`

	// ResponseContentType specifies the HTTP `Content-Type` header that we'll be responding with.
	// This defaults to "application/json".
	ResponseContentType *string `json:"responseContentType,omitempty"`
}

// rejectActionHookDetails contains some fields which are useful when Hook.Action = ActionReject
type rejectActionHookDetails struct {
	// This action also relies on some fields from `respondActionHookDetails`.

	// RejectionErrorCode specifies an error response's error code when Action = ActionReject
	// It's one of the `matrix.Error*` constants or something similar (that list is not exhaustive).
	RejectionErrorCode *string `json:"rejectionErrorCode,omitempty"`

	// RejectionErrorMessage specifies an error response's error message when Action = ActionReject
	RejectionErrorMessage *string `json:"rejectionErrorMessage,omitempty"`
}

// passModifiedRequestActionHookDetails contains some fields which are useful when Hook.Action = ActionPassModifiedRequest
type passModifiedRequestActionHookDetails struct {
	// InjectJSONIntoResponse contains some JSON fields to inject into the original request
	// Required field.
	InjectJSONIntoRequest *map[string]interface{} `json:"injectJSONIntoRequest,omitempty"`

	// InjectHeadersIntoRequest contains a list of headers that will be injected into the original request
	InjectHeadersIntoRequest *map[string]string `json:"injectHeadersIntoRequest,omitempty"`
}

// passModifiedResponseActionHookDetails contains some fields which are useful when Hook.Action = ActionPassModifiedResponse
type passModifiedResponseActionHookDetails struct {
	// InjectJSONIntoResponse contains some JSON fields to inject into the upstream's response
	// Required field.
	InjectJSONIntoResponse *map[string]interface{} `json:"injectJSONIntoResponse,omitempty"`

	// InjectHeadersIntoResponse contains a list of headers that will be injected into the upstream's response
	InjectHeadersIntoResponse *map[string]string `json:"injectHeadersIntoResponse,omitempty"`
}

type Hook struct {
	// An identifier (name) for this hook
	ID string `json:"id,omitempty"`

	EventType string `json:"eventType,omitempty"`

	// MatchRules contains a list of rules that need to match for this hook to be eligible for execution.
	// Of course, the EventType needs to match as well.
	MatchRules []*HookMatchRule `json:"matchRules"`

	// Action specifies what should happen when the hook matches.
	// See the various `Action*` constants.
	Action string `json:"action"`

	// SkipNextHooksInChain tells whether all other hooks in the same execution chain should be skipped.
	// Execution chain means "eligible hooks of this same event type".
	SkipNextHooksInChain bool `json:"skipNextHooksInChain"`

	restActionHookDetails

	respondActionHookDetails

	rejectActionHookDetails

	passModifiedRequestActionHookDetails

	passModifiedResponseActionHookDetails
}

func (me Hook) IsBeforeHook() bool {
	return strings.HasPrefix(me.EventType, "before")
}

func (me Hook) IsAfterHook() bool {
	return strings.HasPrefix(me.EventType, "after")
}

func (me Hook) Validate() error {
	if me.ID == "" {
		return fmt.Errorf("Hook has no id")
	}

	if !util.IsStringInArray(me.EventType, knownEventTypes) {
		return fmt.Errorf("%s is an invalid event type for hook #%s", me.EventType, me.ID)
	}

	if !util.IsStringInArray(me.Action, knownActions) {
		return fmt.Errorf("%s is an invalid action for hook #%s", me.Action, me.ID)
	}

	// We can allow this and it will work (response modification is scheduled to run later on anyway).
	// But it's confusing to define a before hook, which actually runs as an after hook.
	// Better ask people to define it correctly.
	if me.IsBeforeHook() && me.Action == ActionPassModifiedResponse {
		return fmt.Errorf("action=%s cannot be combined with eventType=%s, found in hook #%s", me.Action, me.EventType, me.ID)
	}

	for idx, matchRule := range me.MatchRules {
		err := matchRule.validate()
		if err != nil {
			return fmt.Errorf("Error when validating hook #%s's match rule #%d: %s", me.ID, idx, err)
		}
	}

	// TODO - additional validation logic would be nice to have.
	// The Executor does some, but it might be helpful to catch problems early on (when loading the policy),
	// not when actually executing a hook.

	return nil
}

func (me Hook) MatchesRequest(request *http.Request) bool {
	for _, matchRule := range me.MatchRules {
		if !matchRule.MatchesRequest(request) {
			return false
		}
	}
	return true
}

func (me Hook) String() string {
	return fmt.Sprintf("<Hook #%s (%s @ %s)>", me.ID, me.Action, me.EventType)
}

func ListToChain(hooksList []*Hook) string {
	if len(hooksList) == 0 {
		return "none"
	}

	var ids []string
	for _, hookObj := range hooksList {
		ids = append(ids, fmt.Sprintf("#%s", hookObj.ID))
	}

	return strings.Join(ids, " -> ")
}
