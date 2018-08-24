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

API endpoints:

- [Policy submission endpoint](#policy-submission-endpoint) - `PUT /_matrix/corporal/policy`

- [Policy-provider reload endpoint](#policy-provider-reload-endpoint) - `POST /_matrix/corporal/policy/provider/reload`


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