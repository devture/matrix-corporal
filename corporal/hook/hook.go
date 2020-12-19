package hook

import (
	"devture-matrix-corporal/corporal/util"
	"fmt"
	"net/http"
	"regexp"
)

// restActionHookDetails contains some fields which are useful when Hook.Action is something like ActionConsultRESTServiceURL
type restActionHookDetails struct {
	// RESTServiceURL specifies the URL of the REST service to call when Action = ActionConsultRESTServiceURL
	// Required field.
	RESTServiceURL *string `json:"RESTServiceURL"`

	// RESTServiceRequestMethod specifies the request method to use when making the HTTP request RESTServiceURL
	// If not specified, a "POST" request will be used.
	RESTServiceRequestMethod *string `json:"RESTServiceRequestMethod"`

	// RESTServiceRequestTimeoutMilliseconds specifies how long the HTTP request to RESTServiceURL is allowed to take.
	// If this is not defined, a default timeout value is used (30 seconds at the time of this writing).
	RESTServiceRequestTimeoutMilliseconds *int `json:"RESTServiceRequestTimeoutMilliseconds"`

	// RESTServiceRequestHeaders specifies any request headers that should be sent to the RESTServiceURL when making requests.
	//
	// Example:
	//	RESTServiceRequestHeaders = map[string]string{
	//		"Authorization": "Bearer: SOME_TOKEN",
	//	}
	RESTServiceRequestHeaders *map[string]string `json:"RESTServiceRequestHeaders"`

	// RESTContingencyHook contains a fallback hook to return as a result if the REST service fails.
	//
	// This can both be a communication failure or it returning a response we can't make sense of.
	//
	// If RESTContingencyHook is not defined, any such REST service failures
	// cause execution to stop (503 / "service unavailable").
	RESTContingencyHook *Hook `json:"RESTContingencyHook"`
}

type respondActionHookDetails struct {
	// Payload specifies the payload to respond with.
	// This may be some key-value JSON thing (`map[string]interface{}`), a string, etc.
	ResponsePayload interface{} `json:"responsePayload"`

	// ResponseSkipPayloadJSONSerialization specifies whether the payload found in ResponsePayload should be JSON-serialized.
	// This only applies when ResponseContentType = "application/json".
	// This defaults to false. That is, we serialize to JSON by default (when ResponseContentType = "application/json").
	ResponseSkipPayloadJSONSerialization bool `json:"responseSkipPayloadJSONSerialization"`

	// ResponseStatusCode specifies the HTTP response code that we'll be responding with.
	// Required field.
	ResponseStatusCode *int `json:"responseStatusCode"`

	// ResponseContentType specifies the HTTP `Content-Type` header that we'll be responding with.
	// This defaults to "application/json".
	ResponseContentType *string `json:"responseContentType"`
}

// rejectActionHookDetails contains some fields which are useful when Hook.Action = ActionReject
type rejectActionHookDetails struct {
	// This action also relies on some fields from `respondActionHookDetails`.

	// RejectionErrorCode specifies an error response's error code when Action = ActionReject
	// It's one of the `matrix.Error*` constants or something similar (that list is not exhaustive).
	RejectionErrorCode *string `json:"rejectionErrorCode"`

	// RejectionErrorMessage specifies an error response's error message when Action = ActionReject
	RejectionErrorMessage *string `json:"rejectionErrorMessage"`
}

// passInjectJSONIntoResponseActionHookDetails contains some fields which are useful when Hook.Action = ActionPassInjectJSONIntoResponse
type passInjectJSONIntoResponseActionHookDetails struct {
	// InjectJSONIntoResponse contains some JSON fields to inject into the original response
	// Required field.
	InjectJSONIntoResponse *map[string]interface{} `json:"injectJSONIntoResponse"`

	// InjectHeadersIntoResponse contains a list of headers that will be injected into the original response
	InjectHeadersIntoResponse *map[string]string `json:"injectHeadersIntoResponse"`
}

type Hook struct {
	// An identifier (name) for this hook
	ID string `json:"id"`

	EventType string `json:"eventType"`

	RouteMatchesRegex         *string `json:"routeMatchesRegex"`
	RouteMatchesRegexCompiled *regexp.Regexp

	MethodMatchesRegex         *string `json:"methodMatchesRegex"`
	MethodMatchesRegexCompiled *regexp.Regexp

	Action string `json:"action"`

	restActionHookDetails

	respondActionHookDetails

	rejectActionHookDetails

	passInjectJSONIntoResponseActionHookDetails
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

	err := me.ensureInitialized()
	if err != nil {
		return fmt.Errorf("Error when initializing hook #%s: %s", me.ID, err)
	}

	// TODO - additional validation logic would be nice to have.
	// The Executor does some, but it might be helpful to catch problems early on (when loading the policy),
	// not when actually executing a hook.

	return nil
}

func (me Hook) MatchesRequest(request *http.Request) bool {
	// This would have probably already been executed before,
	// because it's also done as part of hook validation. See Validate().
	err := me.ensureInitialized()
	if err != nil {
		panic(err)
	}

	if me.MethodMatchesRegexCompiled != nil {
		if !me.MethodMatchesRegexCompiled.MatchString(request.Method) {
			return false
		}
	}

	if me.RouteMatchesRegexCompiled != nil {
		if !me.RouteMatchesRegexCompiled.MatchString(request.RequestURI) {
			return false
		}
	}

	return true
}

func (me *Hook) ensureInitialized() error {
	if me.RouteMatchesRegex != nil {
		regex, err := regexp.Compile(*me.RouteMatchesRegex)
		if err != nil {
			return err
		}
		me.RouteMatchesRegexCompiled = regex
	}

	if me.MethodMatchesRegex != nil {
		regex, err := regexp.Compile(*me.MethodMatchesRegex)
		if err != nil {
			return err
		}
		me.MethodMatchesRegexCompiled = regex
	}

	return nil
}

func (me Hook) String() string {
	return fmt.Sprintf("<Hook #%s (%s @ %s)>", me.ID, me.Action, me.EventType)
}
