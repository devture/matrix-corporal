package hook

var (
	// ActionConsultRESTServiceURL is an action which will pass the request to a REST service and decide based on that.
	// See restActionHookDetails for fields related to this action.
	ActionConsultRESTServiceURL = "consult.RESTServiceURL"

	// ActionRespond is an action that outright responds to the request with a specified payload.
	// See respondActionHookDetails for fields related to this action.
	//
	// If you need to reject a request, you'd better use the dedicated "reject" action.
	ActionRespond = "respond"

	// ActionReject is an action that outright rejects the request.
	// See rejectActionHookDetails for fields related to this action.
	//
	// This can also be replaced with the more-capable ActionRespond,
	// but using the "reject" action is simpler and more semantically-correct.
	ActionReject = "reject"

	// ActionPassUnmodified is an action that lets the request pass and returns the response as-is.
	ActionPassUnmodified = "pass.unmodified"

	// ActionPassModifiedRequest is an action that lets the request pass, but first modifies its payload.
	// See passInjectJSONIntoRequestActionHookDetails for fields related to this action.
	ActionPassModifiedRequest = "pass.modifiedRequest"

	// ActionPassModifiedResponse is an action that lets the request pass and then adjusts the JSON response.
	// See passModifiedResponseActionHookDetails for fields related to this action.
	ActionPassModifiedResponse = "pass.modifiedResponse"
)

var knownActions = []string{
	ActionConsultRESTServiceURL,
	ActionRespond,
	ActionReject,
	ActionPassUnmodified,
	ActionPassModifiedResponse,
	ActionPassModifiedRequest,
}
