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
