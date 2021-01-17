# Architecture

`matrix-corporal` does both **reconciliation** and **firewalling** (acting as a gateway to the Matrix server).


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
   |          |                     |            |  Internal REST Auth  | <--|-----+
   |          |                     |            |       handler        |    |     |
   |          |                     |            |----------------------|    |     |
   |          |                     |            |                      |    |     |
   |          |---------------------| TCP 41081  |----------------------|    |     |
   |          |  /_matrix/corporal  | ---------> | HTTP API server      |    |     |
   |          |---------------------|            |----------------------|    |     |
   |          |                     |            |                      |    |     |
   |          |  /_matrix/identity  |            |----------------------|    |     |
   |          |  and others may go  |            | Reconciliation       |----+     |
   |          |  elsewhere          |            |----------------------|    |     |
   |          |                     |            |                      |    |     |
   |          +---------------------+            +----------------------+    |     |
   |                                                                         |     |
   |                                                                         |     |
   |                            +--------------------+                       |     |
   | TCP 8448 (federation API)  |                    | TCP 8008 (client API) |     |
   +--------------------------> |   Matrix Synapse   | <---------------------+     |
                                |                    |                             |
                                |                    |                             |
                                |--------------------|                             |
                                | Shared Secret Auth |                             |
                                | password provider  |                             |
                                |       module       |                             |
                                |--------------------|                             |
                                |      REST Auth     |                             |
                                | password provider  | ----------------------------+
                                |       module       |
                                |--------------------|
                                |                    |
                                +--------------------+
```

Things to note:

- `matrix-corporal` captures all `/_matrix` traffic of the Client-Server API (not the federation port and the Server-Server API), so it can allow/deny or change it

- the federation port (`8448`) must not serve the `client` Matrix APIs. If it does, it would be a way to circumvent `matrix-corporal`'s firewalling. Make sure the federation port only serves the federation API. You can make all that traffic go straight to Matrix Synapse, as `matrix-corporal` doesn't care about it.

- you can have other routes (like `/_matrix/identity`) that are forwarded to other servers (like [ma1sd](https://github.com/ma1uta/ma1sd), etc.). With the proper DNS override configuration in ma1sd, some of these routes can be forwarded back to matrix-corporal (and not to the upstream Synapse server).

- for `matrix-corporal` to work, Matrix Synapse needs to be running with:

  - the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider module installed and configured correctly. This is how `matrix-corporal` obtains an access tokens for its own admin user, which is then used for impersonating other users via the [Synapse-specific admin API for logging in as a user](https://github.com/matrix-org/synapse/blob/develop/docs/admin_api/user_admin_api.rst#login-as-a-user)

  - the [REST Auth](https://github.com/ma1uta/matrix-synapse-rest-password-provider) password provider module installed and configured correctly. This is how Interactive Authentication (initiated by Synapse) manages to get forwarded to `matrix-corporal` so it can perform authentication according to the rules in the policy (see [User Authentication](user-authentication.md)).
