# Matrix Corporal HTTP API

`matrix-corporal`'s behavior can be influenced at runtime through its HTTP API.

To enable the API, you need to use the following `matrix-corporal` configuration:

```json
"HttpApi": {
	"Enabled": true,
	"ListenAddress": "0.0.0.0:41081",
	"AuthorizationBearerToken": "HTTP_API_TOKEN"
}
```

You can then send HTTP API requests to the `/_matrix/corporal/<whatever>` endpoints (see below).

Each request needs to be authenticated by being sent with a `Authorization: Bearer HTTP_API_TOKEN` header.

For each API endpoint, when an error occurs, a [standard Matrix error response](https://matrix.org/docs/spec/client_server/r0.4.0.html#api-standards) will be returned.


API endpoints:

- [Policy fetching endpoint](#policy-fetching-endpoint) - `GET /_matrix/corporal/policy`

- [Policy submission endpoint](#policy-submission-endpoint) - `PUT /_matrix/corporal/policy`

- [Policy-provider reload endpoint](#policy-provider-reload-endpoint) - `POST /_matrix/corporal/policy/provider/reload`

- [User access-token retrieval endpoint](#user-access-token-retrieval-endpoint) - `POST /_matrix/corporal/user/{userId}/access-token/new`

- [User access-token release endpoint](#user-access-token-release-endpoint) - `DELETE /_matrix/corporal/user/{userId}/access-token`


## Policy fetching endpoint

**Endpoint**: `GET /_matrix/corporal/policy`

Regardless of the type of [policy provider](policy-providers.md) being used,
`matrix-corporal` can report what [policy](policy.md) it's currently using over its HTTP API.
This is useful for debugging purposes.

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
http://matrix.example.com/_matrix/corporal/policy
```


## Policy submission endpoint

**Endpoint**: `PUT /_matrix/corporal/policy`

Regardless of the type of [policy provider](policy-providers.md) being used,
`matrix-corporal` can receive a new [policy](policy.md) over its HTTP API.
This is mostly useful with [push-style policy providers](#push-style-policy-providers).

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XPUT \
--data @/some/path/to/policy.json \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
http://matrix.example.com/_matrix/corporal/policy
```


## Policy-provider reload endpoint

**Endpoint**: `POST /_matrix/corporal/policy/provider/reload`

Some [policy providers](policy-providers.md) support periodic reloading. Others don't.

Regardless of that, this API can be used to force `matrix-corporal`'s policy provider to fetch a new policy.

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XPOST \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
http://matrix.example.com/_matrix/corporal/policy/provider/reload
```


## User access-token retrieval endpoint

**Endpoint**: `POST /_matrix/corporal/user/{userId}/access-token/new`

This API endpoint lets you obtain an access token for a specific user.

You can use this access token to access the Matrix API however you see fit.
When done, you can dispose of the access token by calling the [user access-token release API endpoint](#user-access-token-release-endpoint).

Example body payload:

```json
{
	"deviceId": "device id goes here",
	"validitySeconds": 300
}
```

`deviceId` is a required parameter, but may go unused. We attempt to obtain access tokens using an [Admin user login API](https://github.com/matrix-org/synapse/blob/develop/docs/admin_api/user_admin_api.rst#login-as-a-user), which doesn't polute the user's device list.

`validitySeconds` specifies how long the token can be used for. You can omit this parameter to obtain a token which never expires.
Even when using an expiring token (obtained with `validitySeconds`), you're still encouraged to [release it](#user-access-token-release-endpoint).

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XPOST \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
-H 'Content-Type: application/json' \
--data '{"deviceId": "device id goes here", "validitySeconds": 300}' \
http://matrix.example.com/_matrix/corporal/user/@user:example.com/access-token/new
```


## User access-token release endpoint

**Endpoint**: `DELETE /_matrix/corporal/user/{userId}/access-token`

To release a previously-obtained access token for a user, submit a `DELETE` request to the following endpoint.

You are required to submit the access token to delete (release) in the body payload:

```json
{"accessToken": "token goes here"}
```

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XDELETE \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
-H 'Content-Type: application/json' \
--data '{"accessToken": "token goes here"}' \
http://matrix.example.com/_matrix/corporal/user/@user:example.com/access-token
```
