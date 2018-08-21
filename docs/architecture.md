# Architecture

`matrix-corporal` does both **reconciliation** and **firewalling** (acting as a gateway to the Matrix server).

Having it do reconciliation only (and not using it as a gateway in front of the Matrix server) will not work well, as `matrix-corporal` relies on capturing authentication requests (`/login`) to the Matrix server.

```
 Client
   |
   |
   |          +---------------------+            +----------------------+
   |          |    Reverse proxy    |            |   matrix-corporal    |
   | TCP 443  |    (nginx, etc.)    |            |                      |
   +--------> |                     |            |                      |
   |          |---------------------| TCP 41080  |----------------------|
   |          |  /_matrix/*         | ---------> | HTTP Gateway server  |----+
   |          |---------------------|            |----------------------|    |
   |          |                     |            |                      |    |
   |          |---------------------| TCP 41081  |----------------------|    |
   |          |  /_matrix/corporal  | ---------> | HTTP API server      |    |
   |          |---------------------|            |----------------------|    |
   |          |                     |            |                      |    |
   |          |  /_matrix/identity  |            |----------------------|    |
   |          |  and others may go  |            | Reconciliation       |----+
   |          |  elsewhere          |            |----------------------|    |
   |          |                     |            |                      |    |
   |          +---------------------+            +----------------------+    |
   |                                                                         |
   |                                                                         |
   |                            +--------------------+                       |
   | TCP 8448 (federation API)  |                    | TCP 8008 (client API) |
   +--------------------------> |   Matrix Synapse   | <---------------------+
                                |                    |
                                |                    |
                                |--------------------|
                                | Shared Secret Auth |
                                | password provider  |
                                |       module       |
                                |--------------------|
                                |                    |
                                +--------------------+
```

Things to note:

- `matrix-corporal` captures all `/_matrix` traffic on the "HTTP port" (that is, not the federation port), so it can allow/deny or change it

- the federation port (8448) must not serve the `client` Matrix APIs. If it does, it would be a way to circumvent `matrix-corporal`'s firewalling. Make sure the federation port only serves the federation API. You can make all that traffic go straight to Matrix Synapse, as `matrix-corporal` doesn't care about it.

- you can have other routes (like `/_matrix/identity`) that are forwarded to other servers (like [mxisd](https://github.com/kamax-io/mxisd), etc.)

- for `matrix-corporal` to work, Matrix Synapse needs to be running with the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider module installed and configured correctly. This is how `matrix-corporal` manages to obtain access tokens for any user in the system and to make [user authentication](user-authentication.md) work.
