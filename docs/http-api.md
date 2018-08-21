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


## Policy-provider-reloading endpoint

**Endpoint**: `POST /_matrix/corporal/policy/provider/reload`

Some [policy providers](policy-providers.md) support periodic reloading. Others don't.
Regardless of that, you may wish to force `matrix-corporal`'s policy provider to fetch a new policy.

To do that, you can submit a `POST` request to the `/_matrix/corporal/policy/provider/reload` URL.

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XPOST \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
http://matrix.example.com/_matrix/corporal/policy/provider/reload
```

## Policy-receiving endpoint

**Endpoint**: `PUT /_matrix/corporal/policy`

Regardless of the type of [policy provider](policy-providers.md) being used,
`matrix-corporal` can receive new policies over its HTTP API.
This is mostly useful with [push-style policy providers](#push-style-policy-providers).

Submitting a new [policy](policy.md) can be done with a `PUT` request to the `/_matrix/corporal/policy` URL.

Example (using [curl](https://curl.haxx.se/)):

```bash
curl \
-XPUT \
--data @/some/path/to/policy.json \
-H 'Authorization: Bearer HTTP_API_TOKEN' \
http://matrix.example.com/_matrix/corporal/policy
```