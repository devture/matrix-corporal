# User Authentication

A [policy](policy.md) contains various users, which `matrix-corporal` manages.

Each user can be authenticated in a different way.
For some users, passwords can be specified in the policy as plain-text.
For other users, passwords can be specified in the policy as a hash (`md5`, `sha1`, etc.).
For others still, passwords can be avoided in the policy and authentication can happen with a REST API call.

The `authType` field in the user policy (see [user policy fields](policy.md#user-policy-fields)), specifies the authentication method for the given user. The `authCredential` field usually contains the actual password, but may contain some other configuration depending on the authentication type (see below).

If you're curious how `matrix-corporal` makes authentication work behind the scenes, see [How authentication works?](#how-authentication-works) below.


## Plain-text passwords

The simplest (and most insecure) way to specify passwords for users in your policy is to embed the passwords in the policy as plain text.

Here's an example user policy:

```json
{
	"id": "@john:example.com",
	"active": true,
	"authType": "plain",
	"authCredential": "PaSSw0rD",
	"displayName": "John",
	"avatarUri": "https://example.com/john.jpg",
	"joinedCommunityIds": ["+a:example.com"],
	"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"]
}
```


## Hashed passwords

For additional security (or in case you only have a hashed password for your users), you can specify passwords as hashed in your user policy. Example:

```json
{
	"id": "@peter:example.com",
	"active": true,
	"authType": "sha1",
	"authCredential": "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
	"displayName": "Just Peter",
	"avatarUri": "",
	"joinedCommunityIds": ["+b:example.com"],
	"joinedRoomIds": ["!roomB:example.com"]
}
```

The following `authType` hash types are currently supported: `md5`, `sha1`, `sha256`, `sha512`, `bcrypt`.

For all hash types, the `authCredential` field is expected to contain the hashed password.


## External authentication via REST API calls

If you'd rather not have passwords right inside the policy document, you can provide an authentication URL and `matrix-corporal` will send requests over there and read the authentication response.

This way, `matrix-corporal` relies on an external HTTP service to do the actual authentication.

Example user policy:

```json
{
	"id": "@george:example.com",
	"active": true,
	"authType": "rest",
	"authCredential": "https://intranet.example.com/_matrix-internal/identity/v1/check_credentials",
	"displayName": "Georgey",
	"avatarUri": "",
	"joinedCommunityIds": ["+a:example.com", "+b:example.com"],
	"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"]
}
```

Each time the user tries to authenticate with the Matrix server, `matrix-corporal` will make a request to the URL specified in `authCredential`.

The HTTP request payload and response are the same as the ones used by the [HTTP JSON REST Authenticator module](https://github.com/kamax-io/matrix-synapse-rest-auth) for Synapse. It's not like you need to use that authenticator module. In fact, if you're using `matrix-corporal`, you don't. It's just that the request/response syntax is the same.

The HTTP call will be a `POST` request with the following payload body:

```json
{
	"user": {
		"id": "full-mx-id-here",
		"password": "plain-text-password-that-the-user-provided"
	}
}
```

To which your server needs to reply with a JSON response like this:

```json
{
	"auth": {
		"success": true
	}
}
```

.. well, or a `success` value of `false`, if authentication fails.

To test your REST endpoint's implementation, you can send example requests with with [curl](https://curl.haxx.se/) like this:

```bash
curl \
-XPOST \
--data-raw '{"user": {"id": "@user:example.com", "password": "some-password"}}' https://intranet.example.com/_matrix-internal/identity/v1/check_credentials
```

If the HTTP authentication service is down (unreachable or responds with some non-200-OK HTTP status), to prevent downtime, `matrix-corporal` will reuse authentication data from previous authentication sessions. That is, if a given user (say `@user:example.com`) has been found to have authenticated through `matrix-corporal` with a pasword of `some-password` a while ago, that same authentication combination will be allowed until the HTTP authentication service becomes operational again.


## How authentication works?

The Matrix Synapse server only works with `bcrypt` passwords for users.

To make all password providers (as described above) work, we can't possibly store passwords inside Matrix Synapse's database.

Instead, passwords are either stored inside the policy (in the case of [plain-text passwords](#plain-text-passwords) and [hashed passwords](#hashed-passwords)) or delegated to an external service (in the case of [External authentication via REST API calls](#external-authentication-via-rest-api-calls)).

To make all these work, `matrix-corporal` intercepts the authentication endpoint of the client API (something like `/_matrix/client/r0/login`). Once intercepted, the login request is processed in `matrix-corporal`.

Authentication requests for users not managed by `matrix-corporal` (users that do not have a corresponding user policy in the [policy](policy.md)) are directly forwarded to Matrix Synapse -- these users are not managed by `matrix-corporal`, so they are left alone.

If a user is managed by `matrix-corporal`, authentication proceeds depending on the [user authentication](user-authentication.md) type (`authType` user policy field) for the particular user trying to log in.

If the request ends up being **not authenticated**, `matrix-corporal` outright rejects it and it never reaches Matrix Synapse.

If the request ends up being **authenticated**, `matrix-corporal` modifies it (in a way that Matrix Synapse would accept) and forwards it over to Matrix Synapse. The modification part relies on the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) module being enabled in Matrix Synapse. This is how `matrix-corporal` manages to obtain access tokens for any user in the system or create `/login` requests that Matrix Synapse would accept.
