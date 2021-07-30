# Event Hooks

Hooks put you in control of all Client-Server API requests hitting your server.

Since its very beginning, `matrix-corporal` has been capturing some of the Client-Server API routes and checking them against the [policy](policy.md) before letting them pass through to the homeserver or not. These are just a limited number of URL routes and, while very convenient, you were not geting much control over them.

`matrix-corporal` still supports that (it's not going away), but it now also lets you hook into any Client-Server API request & response, similar to what [mxgwd](https://github.com/kamax-matrix/mxgwd) was designed to do.

With the event-hook system, `matrix-corporal` acts like a flexibile firewall.

With hooks, you can:

- **catch requests** before they hit the upstream homeserver (Synapse) or after the response comes and awaits forwarding to the user
- define static rules for **rejecting certain requests**
- define static rules for **modifying certain requests' payload** (want to sanitize content or enforce some rules?)
- define static rules for **modifying certain requests' responses** (want to add some additional fields/headers or override what the homeserver sent?)
- **send the original request to your own REST service** for inspection/logging/modification/rejection (want to use code to tinker with the request?)
- **send the upstream response to your own REST service** for inspection/logging/modification/rejection (want to use code to tinker with the response coming from the homeserver?)

## Example

`policy.json` [policy](policy.md) (partial, just the `hooks` section):

```json
{
	"hooks": [
		{
			"id": "custom-hook-to-prevent-banning-in-all-rooms-except-one",

			"eventType": "beforeAuthenticatedRequest",

			"matchRules": [
				{"type": "method", "regex": "POST"},
				{"type": "route", "regex": "^/_matrix/client/r0/rooms/!some-room-exception:server/ban", "invert": true},
				{"type": "route", "regex": "^/_matrix/client/r0/rooms/!some-room:server/ban"}
			],

			"action": "reject",

			"responseStatusCode": 403,
			"rejectionErrorCode": "M_FORBIDDEN",
			"rejectionErrorMessage": "Banning is forbidden on this server. We're nice like that!"
		},

		{
			"id": "force-every-message-to-say-hello",

			"eventType": "beforeAnyRequest",

			"matchRules": [
				{"type": "route", "regex": "^/_matrix/client/r0/rooms/[^/]+/send/m.room.message/[^/]+$"}
			],

			"action": "pass.modifiedRequest",

			"injectJSONIntoRequest": {
				"body": "Hello!"
			}
		},

		{
			"id": "custom-hook-to-reject-room-creation-once-in-a-while",

			"eventType": "beforeAuthenticatedRequest",

			"matchRules": [
				{"type": "route", "regex": "^/_matrix/client/r0/createRoom"}
			],

			"action": "consult.RESTServiceURL",

			"RESTServiceURL": "http://hook-rest-service:8080/reject/with-33-percent-chance",
			"RESTServiceRequestHeaders": {
				"Authorization": "Bearer SOME_TOKEN"
			},

			"RESTServiceContingencyHook": {
				"action": "reject",
				"responseStatusCode": 403,
				"rejectionErrorCode": "M_FORBIDDEN",
				"rejectionErrorMessage": "REST service down. Rejecting you to be on the safe side"
			}
		},

		{
			"id": "custom-hook-to-capture-and-log-room-creation-details",

			"eventType": "afterAnyRequest",

			"matchRules": [
				{"type": "route", "regex": "^/_matrix/client/r0/createRoom"}
			],

			"action": "consult.RESTServiceURL",

			"RESTServiceURL": "http://hook-rest-service:8080/dump",
			"RESTServiceRequestHeaders": {
				"Authorization": "Bearer SOME_TOKEN"
			},
			"RESTServiceAsync": true,
			"RESTServiceAsyncResultHook": {
				"action": "pass.modifiedResponse",

				"injectJSONIntoResponse": {
					"info": "We're asynchronously logging this /createRoom call and telling you about it here."
				}
			}
		},

		{
			"id": "allow-a-few-users-to-search-the-user-directory",

			"eventType": "beforeAnyRequest",

			"matchRules": [
				{"type": "route", "regex": "^/_matrix/client/r0/user_directory/search"},
				{"type": "matrixUserID", "regex": "^@(george|peter|admin):", "invert": true}
			],

			"action": "pass.unmodified",

			"skipNextHooksInChain": true
		},
		{
			"id": "block-user-directory-searching-for-everyone-else",

			"eventType": "beforeAnyRequest",

			"matchRules": [
				{"type": "route", "regex": "^/_matrix/client/r0/user_directory/search"},
			],

			"action": "reject",

			"responseStatusCode": 403,
			"rejectionErrorCode": "M_FORBIDDEN",
			"rejectionErrorMessage": "Only @george, @peter and @admin can search the user directory. Sorry!"
		}
	],
}
```

The above REST service hooks actually work when you test them in the [development environment](development.md).
They're [implemented in this PHP script](../etc/services/hook-rest-service/index.php).

## Event Types

Event Types are the specific points in the HTTP request/response lifecycle that you can hook into.

The `eventType` field for a given hook can take these values:

- `beforeAnyRequest` - a hook event type which gets executed before requests. This is always executed, for any request URL, be it a known-one to `matrix-corporal` or a catch-all. "Known URLs" (those special to `matrix-corporal`) still get checked against the [policy](policy.md) (as expected), but this hook runs before that happens. This hook fires for all requests, no matter the authentication status.

- `beforeAuthenticatedRequest` - the same as `beforeAnyRequest`, but only gets fired for authenticated requests. If you only you're operating on policy-checked requests, you may be even more specific and use `beforeAuthenticatedPolicyCheckedRequest`.

- `beforeAuthenticatedPolicyCheckedRequest` - a hook event type which gets executed before policy-checking for known URLs. This only gets executed for URLs known and handled by `matrix-corporal` (checked against the policy). This gets triggered before the actual policy-checking. This hook fires only for authenticated requests.

- `beforeUnauthenticatedRequest` - the same as `beforeAnyRequest`, but only gets fired for unauthenticated requests.

- `afterAnyRequest` - a hook event type which gets executed after a request goes through the reverse-proxy, but before its response gets delivered. It allows you to capture (and potentially overwrite) the response coming from the upstream. Say you wish to do something with every room that ever gets created (`/createRoom`). You can set up an `afterAnyRequest` hook and receive the request and response payloads for `/creatRoom` API calls. From there, you can extract the room id, user who did it, etc., and run your own custom logic (e.g. logging, auto-joining others that need to be in that room, etc.). This hook fires for all requests, no matter the authentication status.

- `afterAuthenticatedRequest` - the same as `afterAnyRequest`, but only gets fired for authenticated requests. If you only you're operating on policy-checked requests, you may be even more specific and use `afterAuthenticatedPolicyCheckedRequest`. This hook does not fire for the `/login` route, even if authentication passes successfully. Consider using `afterUnauthenticatedRequest` or `afterAnyRequest` for it.

- `afterUnauthenticatedRequest` - the same as `afterAnyRequest`, but only gets fired for unauthenticated requests.

## Matching rules

Besides matching on **event type**, whether a hook is eligible for running or not depends on a list of matching rules defined in `matchRules`.

You'll most likely wish to perform actions for some URLs (and not for others) and for some HTTP method types (and not for others).

`matchRules` is a list of objects, each of which has a `type` and some more fiels (`regex`, `invert`, etc.)

For a hook to match, all of its match rules need to match (a logical `AND` is applied).

Whether an individual match rule can be inverted by setting `invert` to `true` on it.

Below are the `type` values that we support:


- `type = method` - specifies a regular expression (in the `regex` field) needs to match against the incoming HTTP request's method (GET, POST, etc.).

	Example (matches all `POST` requests, regardless of URL, etc.):

	```json
	{
		"id": "some-hook-id",
		"matchRules": [
			{"type": "method", "regex": "POST"},
		]
	}
	```

- `type = route` - specifies that a regular expression (in the `regex` field) needs to match against the incoming HTTP request's URI.

	Matching with the value found in `regex` is done against the parsed path of the request URI (no query string). Example:
    - original request URI: `/_matrix/client/r0/rooms/!AbCdEF%3Aexample.com/invite?something=here`
    - parsed path: `/_matrix/client/r0/rooms/!AbCdEF:example.com/invite` (this is what matching happens against)

	Example (matches `POST /_matrix/client/r0/createRoom` calls):

	```json
	{
		"id": "some-hook-id",
		"matchRules": [
			{"type": "method", "regex": "POST"},
			{"type": "route", "regex": "^/_matrix/client/r0/createRoom"}
		]
	}
	```

	Example (matches `POST ^/_matrix/client/r0/rooms/../ban` calls, except for one specific room):
	```json
	{
		"id": "some-hook-id",
		"matchRules": [
			{"type": "method", "regex": "POST"},
			{"type": "route", "regex": "^/_matrix/client/r0/rooms/!some-room:example.com/ban", "invert": true},
			{"type": "route", "regex": "^/_matrix/client/r0/rooms/([^/]+)/ban"}
		]
	}
	```

- `type = matrixUserID` - specifies a regular expression (in the `regex` field) needs to match against the full Matrix ID of the user making the request (e.g. `@user:example.com`).

	Example (matches `POST /_matrix/client/r0/createRoom` calls, **not** made by the specified users):
	```json
	{
		"id": "some-hook-id",
		"matchRules": [
			{"type": "method", "regex": "POST"},
			{"type": "route", "regex": "^/_matrix/client/r0/createRoom"},
			{"type": "matrixUserID", "regex": "^@(george|peter|admin):example\.com", "invert": true}
		]
	}
	```

## Actions

After `matrix-corporal` has determined that a given hook is eligible for running (matches the [event type](#event-types) and other [matching rules](#matching-rules)), the next step is actually executing it.

A hook can perform these types of actions (valid values for the `action` field):

  - [Action `pass.unmodified`](#action-passunmodified)
  - [Action `pass.modifiedRequest`](#action-passmodifiedrequest)
  - [Action `pass.modifiedResponse`](#action-passmodifiedresponse)
  - [Action `reject`](#action-reject)
  - [Action `respond`](#action-respond)
  - [Action `consult.RESTServiceURL`](#action-consultrestserviceurl)

### Action `pass.unmodified`

This type of action just makes the request pass through as if it would have originally.

Using this on a hook defined in your [policy](policy.md) file is somewhat useless (why catch a request and then release it?). However, this hook is important when used from within a [`consult.RESTServiceURL` action hook](#action-consultrestserviceurl).

Example:

```json
{
	"id": "my-hook-which-does-nothing-for-each-and-every-URL",

	"eventType": "beforeAnyRequest",

	"action": "pass.unmodified",

	"skipNextHooksInChain": false
}
```

### Action `pass.modifiedRequest`

This type of action makes the request pass through to the upstream homeserver, but modifies its incoming request data (body payload, HTTP headers).

If `action` is set to `pass.modifiedRequest`, you can control execution with the following fields:

- `injectJSONIntoRequest` - a JSON dictionary containing fields to be merged into the original JSON payload. Naturally, this means that you can only use this for modifying JSON payloads, which is what most of the Client-Server APIs take (except for the media repository routes).

- `injectHeadersIntoRequest` (optional) - a JSON dictionary containing a map of header names to header values, which are to be used to modify the original request's HTTP headers.

- `skipNextHooksInChain` (optional, default `false`) - tells whether other matching hooks in the same chain (hooks with the same `eventType`) will be executed

Example:

```json
{
	"id": "request-which-forces-every-sent-message-to-say-hello",

	"eventType": "beforeAnyRequest",

	"matchRules": [
		{"type": "route", "regex": "^/_matrix/client/r0/rooms/[^/]+/send/m.room.message/[^/]+$"}
	],

	"action": "pass.modifiedRequest",

	"injectJSONIntoRequest": {
		"body": "Hello!"
	},
	"injectHeadersIntoRequest": {
		"X-Modified-By-Corporal-Hook": "1"
	},

	"skipNextHooksInChain": false
}
```

### Action `pass.modifiedResponse`

This type of action makes the request pass through to the upstream homeserver, but modifies the resulting HTTP response coming from it (body payload, HTTP headers).

If `action` is set to `pass.modifiedResponse`, you can control execution with the following fields:

- `injectJSONIntoResponse` - a JSON dictionary containing fields to be merged into the response JSON payload (coming from the upstream homeserver). Naturally, this means that you can only use this for modifying JSON response payloads, which is what most of the Client-Server APIs return (except for the media repository routes).

- `injectHeadersIntoResponse` (optional) - a JSON dictionary containing a map of header names to header values, which are to be used to modify the response's HTTP headers.

- `skipNextHooksInChain` (optional, default `false`) - tells whether other matching hooks in the same chain (hooks with the same `eventType`) will be executed

Example:

```json
{
	"id": "inject-field-into-matrix-client-versions",

	"eventType": "afterAnyRequest",

	"matchRules": [
		{"type": "route", "regex": "^/_matrix/client/versions"}
	],

	"action": "pass.modifiedResponse",

	"injectJSONIntoResponse": {
		"homeserverFrontedByCorporal": true
	},

	"skipNextHooksInChain": false
}
```

Modifying responses with a static rule like this is likely not very useful (there's only so much you can do). However, this hook is important when used from within a `consult.RESTServiceURL` hook.

`pass.modifiedResponse` only works with `after*` [event types](#event-types). At the time a `before*` hook runs, there's no response yet. `matrix-corporal` will report this error.

### Action `reject`

This type of action outright rejects a request by responding with some predefined response.

If `action` is set to `reject`, you can control execution with the following fields:

- `responseStatusCode` - the HTTP status code of the rejection response

- `rejectionErrorCode` - the rejection response's `errcode` field (e.g. `M_FORBIDDEN`, `M_NOT_FOUND`, etc.). Itcan be anything, but it may be helpful to stick to the [Matrix Client-Server API's Standards](https://matrix.org/docs/spec/client_server/r0.6.1#api-standards)

- `rejectionErrorMessage` - a more user-friendly error message describing why the rejection happened. Ends up in the response's `error` field.

Example:

```json
{
	"id": "custom-hook-to-prevent-banning",

	"eventType": "beforeAnyRequest",

	"matchRules": [
		{"type": "method", "regex": "POST"},
		{"type": "route", "regex": "^/_matrix/client/r0/rooms/([^/]+)/ban"}
	],

	"action": "reject",

	"responseStatusCode": 403,
	"rejectionErrorCode": "M_FORBIDDEN",
	"rejectionErrorMessage": "Banning is forbidden on this server. We're nice like that!"
}
```

The above example produces the following `Content-Type: application/json` response (with an HTTP status of `403`):

```json
{
	"errcode": "M_FORBIDDEN",
	"error": "Banning is forbidden on this server. We're nice like that!"
}
```

For even more flexibility when responding, consider using the `respond` action instead.

You can use `reject` actions on both `before*` and `after*` [event type](#event-types) hooks,
depending on whether you wish for the rejection to happen before or after it hits the upsteam homeserver.
Rejecting after it hits it likeky has fewer applications (if it has any at all).

### Action `respond`

This type of action outright responds to a request with some predefined response. It's like `reject`, but more flexible and meant to be used for non-rejection responses.

If `action` is set to `respond`, you can control execution with the following fields:

- `responseStatusCode` - the HTTP status code of the response
- `responseContentType` (default `application/json`) - the `Content-Type` header of the response you're sending (defaults to `application/json`)
- `responsePayload` - the payload to respond with, specified either as a JSON dictionary or string. Specifiying it as a string can be confusing (do you wish to respond with a string value, or does that string value contain parseable JSON). If you'd like to actually send JSON while defining the payload as a string here, consider using `responseSkipPayloadJSONSerialization = true` as well.
- `responseSkipPayloadJSONSerialization` (default `false`) - specifies whether the payload should *skip* being serialized as JSON and instead attempted to be delivered directly (as-is). If `responsePayload` contains a *string* containing parseable JSON, you likely wish to set `responseSkipPayloadJSONSerialization` to `true`.

Example:

```json
{
	"id": "capture-displayname-change-attempts-and-pretend-to-accept-them",

	"eventType": "beforeAnyRequest",

	"matchRules": [
		{"type": "method", "regex": "PUT"},
		{"type": "route", "regex": "^/_matrix/client/r0/profile/([^/]+)/displayname"}
	],

	"action": "respond",

	"responseStatusCode": 200,
	"responsePayload": {}
}
```

The above example produces the following `Content-Type: application/json` response (with an HTTP status of `200`):

```json
{}
```

### Action `consult.RESTServiceURL`

This type of action makes a call to your own REST service URL, which could inspect the request (and response, for `after*` hooks) and then, in turn, respond with another action.

While all other hooks were just static rules, this one is very powerful, as it gives you programatic control over what happens with a request or request, either before or after contacting the upstream server.

If `action` is set to `consult.RESTServiceURL`, you can control execution with the following fields:

- `RESTServiceURL` - specifies the URL that should be consulted

- `RESTServiceRequestMethod` (default `POST`) - specifies the HTTP request method that `RESTServiceURL` is contacted with.

- `RESTServiceRequestHeaders` (default `{}`) - specifies a dictionary of header names and header values, to be sent to your `RESTServiceURL`. You can use this to send some authentication data (e.g. `Authorization` header with some value like `Bearer TOKEN_HERE`, etc), so that your REST service can trust that it's really `matrix-corporal` that is calling it.

- `RESTServiceRequestTimeoutMilliseconds` (default `30`) - specifies how long the HTTP request to `RESTServiceURL` is allowed to take.

- `RESTServiceRetryAttempts` (default `0`) - specifies how many times to retry the REST service HTTP request if failures are encountered. If not specified, no retries will be attempted.

- `RESTServiceRetryWaitTimeMilliseconds` (default `0`) - specifies how long to wait between retries when contacting the REST service. This only makes sense if `RESTServiceRetryAttempts` is set to a positive number. If not specified, retries will happen immediately without waiting.

- `RESTServiceAsync` (default `false`) - specifies whether REST HTTP calls should be waited upon. If not specified, we default to waiting on them and extracting their result (a new hook object). If this is set to `true`, we'll simply fire the request and not care about what the response is. We'll still retry (obeying `RESTServiceRetryAttempts` and `RESTServiceRetryWaitTimeMilliseconds`) and expect an OK (200) response, but it will no longer block the request, nor can it influence it. The result of async REST hooks can be specified in `RESTServiceAsyncResultHook`. By default (if not specified), we let the original request/response pass through unmodified.

- `RESTServiceAsyncResultHook` (default `{"action": "pass.unmodified"}`) - specifies the result for *async* hooks (`RESTServiceAsync = true`). Because we don't wait upon these hooks to actually return a resulting hook, yet wish to know what to do next, we ask you to define the next action here, defaulting to "doing nothing".

- `RESTServiceContingencyHook` (default `null`) - specifies a contingency plan hook for what should be done, if REST service consultation ultimately fails. By default, no contingency hook is defined and we'll return a `503` internal server error response. Using this, you can specify an alternative. You can fall back to any other action, including another `consult.RESTServiceURL` call.

Your REST service URL **must** respond with an HTTP status code of exactly `200`. Other OK-ish response statuses (`201`, `204`, etc.) are not considered a successful execution and will result in a retry attempt (if retries configured) and ultimately a failure.

Because this hook relies on an external REST service, processing failures are more likely.
We let you control the timeout and retries, as a way to minimize the damage.

If consulting the REST service ultimately fails, we let you fall back to a contingency hook (executing some other action instead).

If there's no contingency hook defined and a failure occurs (for *synchronous* REST hooks), we play it safe and abort the request/response lifecycle.

Example:

```json
{
	"id": "custom-hook-to-reject-room-creation-once-in-a-while",

	"eventType": "beforeAuthenticatedPolicyCheckedRequest",

	"matchRules": [
		{"type": "method", "regex": "POST"},
		{"type": "route", "regex": "^/_matrix/client/r0/createRoom"}
	],

	"action": "consult.RESTServiceURL",

	"RESTServiceURL": "http://hook-rest-service:8080/reject/with-33-percent-chance",
	"RESTServiceRequestHeaders": {
		"Authorization": "Bearer SOME_TOKEN"
	},

	"RESTServiceRequestTimeoutMilliseconds": 3000,
	"RESTServiceRetryAttempts": 3,
	"RESTServiceRetryWaitTimeMilliseconds": 1000,

	"RESTServiceContingencyHook": {
		"action": "reject",
		"responseStatusCode": 403,
		"rejectionErrorCode": "M_FORBIDDEN",
		"rejectionErrorMessage": "REST service down. Rejecting you to be on the safe side"
	},

	"skipNextHooksInChain": false
}
```

Example JSON payload that hits your REST service:

```json
{
	"meta": {
		"hookId": "custom-hook-to-reject-room-creation-once-in-a-while",
		"authenticatedMatrixUserId":"@a:matrix-corporal.127.0.0.1.nip.io"
	},

	"request": {
		"URI": "/_matrix/client/r0/createRoom",
		"path": "/_matrix/client/r0/createRoom",
		"method": "POST",
		"headers": {
			"Accept": "application/json",
			"Accept-Encoding": "gzip, deflate",
			"Accept-Language": "en-US,en;q=0.5",
			"Authorization": "Bearer ACCESS_TOKEN_HERE",
			"Connection": "keep-alive",
			"Content-Length": "197",
			"Content-Type": "application/json",
			"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0"
		},
		"payload": "{\"name\":\"Room name\",\"preset\":\"private_chat\",\"visibility\":\"private\",\"initial_state\":[{\"type\":\"m.room.guest_access\",\"state_key\":\"\",\"content\":{\"guest_access\":\"can_join\"}}]}"
	},

	"response": {
		"statusCode": 200,
		"headers": {
			"Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept, Authorization, Date",
			"Access-Control-Allow-Methods": "GET, HEAD, POST, PUT, DELETE, OPTIONS",
			"Access-Control-Allow-Origin": "*",
			"Cache-Control": "no-cache, no-store, must-revalidate",
			"Content-Type": "application/json",
			"Date": "Sat, 16 Jan 2021 19:23:08 GMT",
			"Server": "Synapse/1.25.0"
		},
		"payload":"{\"room_id\":\"!zoFOpIhxSyiJDqXCqv:matrix-corporal.127.0.0.1.nip.io\"}"
	}
}
```

You'll only get a `response` field if your REST service gets called for an `after*` hook.

Example reply you may send:

```json
{
	"action": "pass.unmodified",

	"skipNextHooksInChain": false
}
```

The above REST service hook actually work when you test it in the [development environment](development.md).
It's [implemented in this PHP script](../etc/services/hook-rest-service/index.php).


## Execution notes

The event types differ depending on the route and the user-authentication state - we don't run `{before,after}AuthenticatedRequest` hooks for unauthenticated users.

Some types of `eventType` + `action` combinations make no sense. `matrix-corporal` will tell you about it.
For example, trying to do `action = pass.modifiedRequest` from an `after*` hook makes no sense. At the time `after*` hooks run, it's already too late to modify the request (it has already been sent to the upstream server, a response has arrived, etc.).

`matrix-corporal` runs **all matching hooks** that match a given request.

If you define 2 `pass.modifiedRequest` hooks that match the request, both will be executed, in order.

If you'd like to break the execution flow, you can make one of these hooks set `skipNextHooksInChain` to `true`,
or you can introduce a no-op hook between them, which consists of `action = pass.unmodified` and `skipNextHooksInChain = true`.
