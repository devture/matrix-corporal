## Matrix Corporal HTTP Gateway

`matrix-corporal` works by:

- controlling Matrix Synapse (**reconciliation**) to make it match a given [policy](policy.md); and by

- **firewalling** Matrix Synapse, so that certain requests are not allowed to reach it, or are made to reach it with a different payload (to assist logging in, etc.)

As described and illustrated in [Architecture](architecture.md), `matrix-corporal`'s HTTP server is supposed to catch all `/_matrix` traffic (besides the federation traffic, which usually runs on the `8448` federation port).

Requests that `matrix-corporal` does not know or care about are forwarded directly to the upstream homeserver (Synapse).

Requests that `matrix-corporal` is interested in are intercepted and allowed/denied or modified.
Most request are merely allowed/denied, but certain things like [user authentication](user-authentication.md) rely on modifying requests before sending them over to the Matrix server.
