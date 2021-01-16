package hook

// `before*` hooks are executed in the order they're defined below.
// Of course, some only apply to policy-checked routes, some only for authenticated users, etc,
// so it varies depending on what we're working with.
var (
	// EventTypeBeforeAnyRequest is a hook event type which gets executed before requests.
	//
	// This is always executed, for any request URL, be it a known-one to matrix-corporal or a catch-all.
	// "Known URLs" also get checked against the matrix-corporal policy (as expected), but this hook runs before that happens.
	//
	// This hook fires for all requests, no matter the authentication status.
	EventTypeBeforeAnyRequest = "beforeAnyRequest"

	// EventTypeBeforeAuthenticatedRequest is the same as EventTypeBeforeAnyRequest, but only gets fired for authenticated requests.
	//
	// If you only you're operating on policy-checked requests, you may be even more specific
	// and use EventTypeBeforeAuthenticatedPolicyCheckedRequest.
	EventTypeBeforeAuthenticatedRequest = "beforeAuthenticatedRequest"

	// EventTypeBeforeAuthenticatedPolicyCheckedRequest is a hook event type which gets executed before policy-checking for known URLs.
	//
	// This only gets executed for URLs known and handled by matrix-corporal (checked against the policy).
	// This gets triggered before the actual policy-checking.
	//
	// This hook fires only for authenticated requests.
	EventTypeBeforeAuthenticatedPolicyCheckedRequest = "beforeAuthenticatedPolicyCheckedRequest"

	// EventTypeBeforeUnauthenticatedRequest is the same as EventTypeBeforeAnyRequest, but only gets fired for unauthenticated requests.
	EventTypeBeforeUnauthenticatedRequest = "beforeUnauthenticatedRequest"
)

// `after*` hooks are executed in the order they're defined below.
// Of course, some only apply to policy-checked routes, some only for authenticated users, etc,
// so it varies depending on what we're working with.
//
// All `after*` hooks are executed AFTER the request had been forwarded to the reverse-proxy
// and we've received the response from it.
//
// After-hooks allow for this response to be operated on.
var (
	// EventTypeAfterAnyRequest is a hook event type which gets executed after a request goes through the reverse-proxy, but before its response gets delivered.
	//
	// It allows you to capture (and potentially overwrite) the response coming from the upstream.
	//
	// Say you wish to do something with every room that ever gets created (`/createRoom`).
	// You can set up an EventTypeAfterAnyRequest hook and receive the request and response payloads for `/creatRoom` API calls.
	// From there, you can extract the room id, user who did it, etc., and run your own custom logic
	// (e.g. logging, auto-joining others that need to be in that room, etc.)
	//
	// This hook fires for all requests, no matter the authentication status.
	EventTypeAfterAnyRequest = "afterAnyRequest"

	// EventTypeAfterAuthenticatedRequest is the same as EventTypeAfterAnyRequest, but only gets fired for authenticated requests.
	//
	// If you only you're operating on policy-checked requests, you may be even more specific
	// and use EventTypeAfterAuthenticatedPolicyCheckedRequest.
	//
	// This hook does not fire for the `/login` route, even if authentication passes successfully.
	// Consider using EventTypeAfterUnauthenticatedRequest or EventTypeAfterAnyRequest.
	EventTypeAfterAuthenticatedRequest = "afterAuthenticatedRequest"

	// EventTypeAfterAuthenticatedPolicyCheckedRequest is a hook event type which gets executed after a request and only for policy-checked requests.
	//
	// This only gets executed for URLs known and handled by matrix-corporal (checked against the policy).
	// This gets triggered after the actual policy-checking and after the response
	//
	// This hook fires only for authenticated requests.
	//
	// This hook does not fire for the `/login` route, even if authentication passes successfully.
	// Consider using EventTypeAfterUnauthenticatedRequest or EventTypeAfterAnyRequest.
	EventTypeAfterAuthenticatedPolicyCheckedRequest = "afterAuthenticatedPolicyCheckedRequest"

	// EventTypeAfterUnauthenticatedRequest is the same as EventTypeAfterAnyRequest, but only gets fired for unauthenticated requests.
	EventTypeAfterUnauthenticatedRequest = "afterUnauthenticatedRequest"
)

var knownEventTypes = []string{
	EventTypeBeforeAnyRequest,
	EventTypeBeforeAuthenticatedRequest,
	EventTypeBeforeAuthenticatedPolicyCheckedRequest,
	EventTypeBeforeUnauthenticatedRequest,

	EventTypeAfterAnyRequest,
	EventTypeAfterAuthenticatedRequest,
	EventTypeAfterAuthenticatedPolicyCheckedRequest,
	EventTypeAfterUnauthenticatedRequest,
}
