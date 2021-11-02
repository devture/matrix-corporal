# User Authentication

A [policy](policy.md) contains various users, which `matrix-corporal` manages.

Each user can be authenticated in a different way:
- using a passwords specified in the policy as plain-text. See [Plain-text passwords](#plain-text-passwords)
- by using an initial plain-text password specified in the policy, but then delegating password management to the homeserver. See [Passthrough authentication](#passthrough-authentication)
- using a password specified in the policy as a hash (`md5`, `sha1`, etc.). See [Hashed passwords](#hashed-passwords)
- by not specifying a password in the policy, but rather delegating authentication to some REST API. See [External authentication via REST API calls](#external-authentication-via-rest-api-calls)

The `authType` field in the user policy (see [user policy fields](policy.md#user-policy-fields)), specifies the authentication method for the given user. The `authCredential` field usually contains the actual password, but may contain some other configuration depending on the authentication type (see below).

If you're curious how `matrix-corporal` makes authentication work behind the scenes, see [How authentication works?](#how-authentication-works) below.


## Plain-text passwords

The simplest (and most insecure) way to specify passwords for users in your policy is to embed the passwords in the policy as plain text.

Here's an example policy:

```json
{
	"users": [
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
	]
}
```

Users are created on the homeserver with a long-random password. Still, authentication is intercepted by matrix-corporal and password-matching is performed against the **current** password found in `authCredential`.


## Passthrough authentication

A variation of [Plain-text passwords](#plain-text-passwords) is the `passthrough` authentication type.

It's similar to plain-text authentication, but:

- actually creates users on the homeserver with the plain-text password provided in `authCredential` (as opposed to a random long password)
- authentication is not handled in `matrix-corporal` (as with all other auth types), but is instead forwarded to the homeserver and happens against the password stored there
- users **may** be allowed to change their password stored on the homeserver, depending on the `allowCustomPassthroughUserPasswords` flag in the **main** policy (defaults to `false`). See [Policy Flags](policy.md#flags)

Here's an example policy:

```json
{
	"flags": {
		"allowCustomPassthroughUserPasswords": true
	},

	"users": [
		{
			"id": "@john:example.com",
			"active": true,
			"authType": "passthrough",
			"authCredential": "some-initial-password",
			"displayName": "John",
			"avatarUri": "https://example.com/john.jpg",
			"joinedCommunityIds": ["+a:example.com"],
			"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"]
		}
	]
}
```

Note that subsequent changes to the `authCredential` value in the policy do not update the homeserver password for the user. That is, `authCredential` only serves as an initial password, which get be changed later, independently of `matrix-corporal`.


## Hashed passwords

For additional security (or in case you only have a hashed password for your users), you can specify passwords as hashed in your user policy.

Here's an example policy:

```json
{
	"users": [
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
	]
}
```

The following `authType` hash types are currently supported: `md5`, `sha1`, `sha256`, `sha512`, `bcrypt`.

For all hash types, the `authCredential` field is expected to contain the hashed password.


## External authentication via REST API calls

If you'd rather not have passwords right inside the policy document, you can provide an authentication URL and `matrix-corporal` will send requests over there and read the authentication response.

This way, `matrix-corporal` relies on an external HTTP service to do the actual authentication.

Here's an example policy:

```json
{
	"users": [
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
	]
}
```

Each time the user tries to authenticate with the Matrix server, `matrix-corporal` will make a request to the URL specified in `authCredential`.

The HTTP request payload and response are the same as the ones used by the [HTTP JSON REST Authenticator module](https://github.com/ma1uta/matrix-synapse-rest-password-provider) for Synapse. It's not like you need to use that authenticator module. In fact, if you're using `matrix-corporal`, you don't. It's just that the request/response syntax is the same.

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

An example implementation of the authentication service is available in [`etc/services/rest-password-auth-service/index.php`](../etc/services/rest-password-auth-service/index.php).

If the HTTP authentication service is down (unreachable or responds with some non-200-OK HTTP status), to prevent downtime, `matrix-corporal` will reuse authentication data from previous authentication sessions. That is, if a given user (say `@user:example.com`) has been found to have authenticated through `matrix-corporal` with a password of `some-password` a while ago, that same authentication combination will be allowed until the HTTP authentication service becomes operational again.


## How authentication works?

The Synapse server only works with `bcrypt` passwords for users.

To make all password providers (as described above) work, we can't possibly store passwords inside Synapse's database.

Instead, passwords are either stored inside the policy (in the case of [plain-text passwords](#plain-text-passwords) and [hashed passwords](#hashed-passwords)) or delegated to an external service (in the case of [External authentication via REST API calls](#external-authentication-via-rest-api-calls)).

To make all these work, `matrix-corporal` intercepts the authentication endpoint of the client API (something like `/_matrix/client/r0/login`). Once intercepted, the login request is processed in `matrix-corporal`.

Authentication requests with a login flow of `m.login.token` (used by CAS/SAML SSO login) are directly forwarded to the upstream server unchanged.

Authentication requests for users not managed by `matrix-corporal` (users that do not have a corresponding user policy in the [policy](policy.md)) are directly forwarded to the upstream server -- these users are not managed by `matrix-corporal`, so they are left alone.

Requests for users having `authType=passthrough` are forwarded to the upstream server unchanged.

For requests for users having another auth type (different than `passthrough`), authentication proceeds depending on the [user authentication](user-authentication.md) type (`authType` user policy field) for the particular user trying to log in.

If the request ends up being **not authenticated**, `matrix-corporal` outright rejects it and it never reaches the upstream server.

If the request ends up being **authenticated**, `matrix-corporal` modifies it (in a way that the upstream server would accept) and forwards it over to the upstream server. The modification part relies on the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) module being enabled in Synapse. This is how `matrix-corporal` manages to obtain access tokens for any user in the system or create `/login` requests that Synapse would accept.
