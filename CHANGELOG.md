# Version 2.2.3 (2022-01-31)

Internal compiler and dependency upgrades.

# Version 2.2.2 (2021-12-01)

Internal compiler and dependency upgrades.

# Version 2.2.1 (2021-11-20)

Internal compiler and dependency upgrades.

# Version 2.2.0 (2021-11-19)

Adds support for handling `v3` Client-Server API requests, instead of rejecting them as unknown.
Synapse v1.48.0 is meant to [add support](https://github.com/matrix-org/synapse/pull/11318) for v3 APIs (as per [Matrix Spec v1.1](https://matrix.org/blog/2021/11/09/matrix-v-1-1-release)).

We patched a what-would-become a security vulnerability related to this in matrix-corporal 2.1.5. Read below.

The matrix-corporal 2.2.0 release continues the v3 work by actually handling v3 requests (the same way r0 requests are handled).


# Version 2.1.5 (2021-11-19)

Fixes an issue which would become a security vulnerability starting with Synapse v1.48.0 (to be released in the future).

Synapse v1.48.0 is meant to [add support](https://github.com/matrix-org/synapse/pull/11318) for v3 APIs (as per [Matrix Spec v1.1](https://matrix.org/blog/2021/11/09/matrix-v-1-1-release)).
`/_matrix/client/v3/..` requests could circuimvent matrix-corporal's policy checks, because it only handled the `r0` Client-Server API version (as well as other `r`-prefixed versions).

The `v`-prefixed naming scheme was not supported by matrix-corporal until now, so such requests could go through unchecked.
Running the upcoming Synapse v1.48.0+ release with matrix-corporal (`<2.1.4`) would become a security issue, so it's important to update to matrix-corporal 2.1.5.

More complete `v3` support will be added to matrix-corporal in a future release (matrix-corporal 2.2.0).

# Version 2.1.4 (2021-11-15)

Fixes a regression introduced in 2.1.3, which broke `GET /_matrix/client/r0/pushrules/` requests.

The security fix implemented in 2.1.3 stripped trailing slashes from request URLs. This worked well for most requests,
but broke certain special requests like the one mentioned above.

2.1.4 basically implements the fix found in 2.1.3 in a more robust way.

# Version 2.1.3 (2021-11-15)

Fixes a security-vulnerability, which allowed attackers to circuimvent policy-checks by sending HTTP requests with a trailing slash.

The issue has been discovered accidentally, due to element-web (v1.9.4) sending room state-change requests with a trailing slash like this: `/_matrix/client/r0/rooms/{roomId}/state/m.room.encryption/`. Other policy-checked routes are probably affected just the same, but exploiting this vulnerability only happened with more intentional targeting, rather than accidentally.

# Version 2.1.2 (2021-08-23)

Internal compiler and dependency upgrades.

# Version 2.1.1 (2021-07-10)

Minor changes to match Synapse v1.38.0's CORS behavior. Internal compiler and dependency upgrades.

# Version 2.1.0 (2021-01-18)

This release introduces a new global [policy flag](docs/policy.md#flags) (`flags.allowUnauthenticatedPasswordResets`), which you can use to control whether an unauthenticated password-reset flow (via `/_matrix/client/r0/account/password`) is allowed to happen.

Previously, we were always refusing such non-authenticated requests, but certain servers may wish to allow them.


# Version 2.0.1 (2021-01-17)

Bugfix release for the "Internal REST Auth" feature used for supporting Interactive Authentication, in coordination with [matrix-synapse-rest-password-provider](https://github.com/ma1uta/matrix-synapse-rest-password-provider).


# Version 2.0.0 (2021-01-17)

This is a very large release (hence the version bump) with the following **small breaking changes**:

- `Reconciliation.UserId` configuration key got moved to `Corporal.UserID`
- we now **require** Synapse `>= v1.24.0`. To stay on older versions, use v1 of `matrix-corporal`.
- you're not required to, but may wish to install [matrix-synapse-rest-password-provider](https://github.com/ma1uta/matrix-synapse-rest-password-provider) and point it at `matrix-corporal`. See why below.

The major changes are described below.

## Event Hooks system

We now have `before*` and `after*` event hooks, so `matrix-corporal` can **act like a more generic firewall** (like [mxgwd](https://github.com/kamax-matrix/mxgwd)) - inspecting, modifying and blocking any kind of Matrix Client-Server API request.

Learn more on the [Event Hooks](docs/event-hooks.md) documentation page.

## Going device-free

We now use a [Synapse-specific admin API for logging in as a user](https://github.com/matrix-org/synapse/blob/develop/docs/admin_api/user_admin_api.rst#login-as-a-user) (implemented in Synapse v1.24.0, [here](https://github.com/matrix-org/synapse/pull/8617)).

Until now we were relying on the [matrix-synapse-shared-secret-auth](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider for impersonating users. With that, we were creating login sessions (and devices) that were publicly visible to the user itself and to other users. This could even become slow over federation, because new devices are advertised to everyone you're in contact with.

The new API we use for impersonating users is Synapse specific, but leads to better performance (**reconciliation times are way faster now**, because we don't create useless devices that potentially get advertised over federation). This is also better in terms of resilience and for UX.

Our [User access-token retrieval HTTP API endpoint](docs/http-api.md#user-access-token-retrieval-endpoint) now also obtains access tokens without creating unnecessary devices for users. The API also takes an optional `validitySeconds` parameter allowing you to obtain time-limited tokens.

## Support for Interactive Auth (E2EE-friendly, etc.)

Because of the way we were doing authentication before (capturing `/login` requests and handling it all inside of `matrix-corporal`), we couldn't support Interactive Authentication (initiated by Synapse).

Thanks to `matrix-corporal`'s new "Internal REST Auth" feature, combined with [matrix-synapse-rest-password-provider](https://github.com/ma1uta/matrix-synapse-rest-password-provider), **Interactive Authentication now works**.

To enable it, set `HttpGateway.InternalRESTAuth.Enabled` to `true` and install the REST auth password provider in Synapse, pointing it to `matrix-corporal` (e.g. `http://matrix-corporal:41080/_matrix/corporal`).

Interactive Authentication is required for certain actions that the user performs, such as setting up End-to-End-Encryption (E2EE) keys, managing devices, etc.

Now that we've made it work, `matrix-corporal` is **finally E2EE-friendly**.

## In control of E2EE

Not only is `matrix-corporal` now E2EE-friendly, it can also **enforce** whether rooms that users create are **encrypted or unencrypted**.

That is, if you'd like to force users to only create encrypted rooms, you can. If you'd like to force them to only create unencrypted rooms, you also can. It's up to you.

This is controlled by [global and user-policy flags](docs/policy.md).

## Other minor changes

- fixes a user-creation bug that occurred with Synapse v1.24.0 due to the removal of `/_matrix/client/*/admin` API endpoints (they now live at `/_synapse/admin/*`)

- ability to control how often access tokens are mapped to user IDs (see the `UserMappingResolver` [configuration](docs/configuration.md)). By default, we expire resolver results after 5 minutes (previously never).


# Version 1.12.0 (2021-01-17)

This version fixes a user-creation bug that occurred with Synapse v1.24.0 due to the removal of `/_matrix/client/*/admin` API endpoints (they now live at `/_synapse/admin/*`).


# Version 1.11.0 (2020-10-01)

This version adds support for `authType=passthrough` user authentication.
Learn more from the [User Authentication documentation](docs/user-authentication.md).


# Version 1.10.0 (2020-09-22)

We now use `/_synapse/admin/v2/users` for fetching the list of users on the server (and not `/_matrix/client/r0/admin/users/{userId}`).

The latter should still work for [Synapse v1.20.0](https://github.com/matrix-org/synapse/releases/tag/v1.20.0), but using the newer API is more future-proof.


# Version 1.9.0 (2020-04-17)

Users can now be prevented from creating rooms (that is, matrix-corporal can restrict the `/createRoom` API).
See the new `forbidRoomCreation` [policy](docs/policy.md) fields.


# Version 1.8.0 (2020-03-24)

The HTTP Gateway and HTTP API servers no longer obey `Matrix.TimeoutMilliseconds`,
but rather have their own explicit timeout settings (`HttpGateway.TimeoutMilliseconds` and `HttpApi.TimeoutMilliseconds`).

You'll need to update your configuration to define these settings.
A large value is recommended for `HttpGateway.TimeoutMilliseconds` (at least the same as or larger than `Matrix.TimeoutMilliseconds`).


# Version 1.7.2 (2020-03-24)

The HTTP Gateway and HTTP API servers no longer use a hardcoded timeout value of 15 seconds,
but rather obey `Matrix.TimeoutMilliseconds`, thus fixing a problem where long-running
`/sync` requests were terminated prematurely.


# Version 1.7.1 (2019-12-04)

`/login` requests now respond with `M_USER_DEACTIVATED` for inactive users, instead of `M_FORBIDDEN`.


# Version 1.7.0 (2019-12-03)

`/login` requests now support the new `identifier.user` payload parameter, not just the deprecated `user` parameter.


# Version 1.6.0 (2019-09-24)

`m.login.token` requests to `/login` are no longer denied, but rather passed through to the upstream server (Synapse).
This is done to prevent any potentially-enabled SSO (CAS or SAML) login flows from breaking.


# Version 1.5.0 (2019-08-20)

Various dependencies were updated and code has been refactored a bit.
There are no functionality changes, but the internal refactoring justifies a version bump.


# Version 1.4.0 (2019-04-06)

Building is now based on Go modules, not on the [gb](https://getgb.io/) tool.
Go 1.12 or later is required.


# Version 1.3.0 (2019-01-25)

Reconciliation is now much faster, due to the way we retrieve account data from the Matrix server (no longer doing `/sync`).

From now on, the minimum requirement for running matrix-corporal is Synapse v0.34.1,
as it's the first Synapse release which contains the new API we require (`GET /user/{user_id}/account_data/{account_dataType}`).


# Version 1.2.2 (2018-12-21)

- HTTP gateway: reverse-proxying requests to Synapse now respects the timeout configuration (`Matrix.TimeoutMilliseconds`) and logs errors in a better way


# Version 1.2.1 (2018-09-20)

- HTTP gateway: unified log message format (all messages are prefixed by `HTTP gateway:` now)

- HTTP gateway: added `/_matrix/client/corporal` route to allow for detection/monitoring


# Version 1.2 (2018-09-15)

- HTTP API: returning [standard Matrix error responses](https://matrix.org/docs/spec/client_server/r0.4.0.html#api-standards) when errors occur, instead of the custom `{"ok": false, "error": "Message"}` responses we had until now

- upgraded dependency libraries


# Version 1.1.1 (2018-09-08)

- Reconciliation: speeding up account-data fetching by optimizing the /sync call

- Upgrading Go compiler (1.10 -> 1.11)


# Version 1.1 (2018-08-24)

- HTTP API: improved logging support

- HTTP API: added 2 new endpoints: [User access-token retrieval](docs/http-api.md#user-access-token-retrieval-endpoint) and [User access-token release](docs/http-api.md#user-access-token-release-endpoint)


# Version 1.0 (2018-08-21)

Initial release.
