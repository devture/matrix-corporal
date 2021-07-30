# Matrix Corporal Configuration

The `matrix-corporal` configuration is a JSON document that looks like this:

```json
{
	"Matrix": {
		"HomeserverDomainName": "matrix-corporal.127.0.0.1.nip.io",
		"HomeserverApiEndpoint": "http://matrix-corporal.127.0.0.1.nip.io:41408",
		"AuthSharedSecret": "7DXvheK1400ydCHjAymDU50FkeUedQJ2AYpitr3inLpSBIdRJN4kfS5IkGYvUptF",
		"RegistrationSharedSecret": "y4aTYam;zxKZ#MnaHRrGDPs4&dS*3VEv_&Ck_;pe1=CrtM8*=7",
		"TimeoutMilliseconds": 45000
	},

	"Corporal": {
		"UserId": "@matrix-corporal:matrix-corporal.127.0.0.1.nip.io"
	},

	"Reconciliation": {
		"RetryIntervalMilliseconds": 30000
	},

	"HttpGateway": {
		"ListenAddress": "127.0.0.1:41080",
		"TimeoutMilliseconds": 60000,
		"InternalRESTAuth": {
			"Enabled": true
		}
	},

	"HttpApi": {
		"Enabled": true,
		"ListenAddress": "127.0.0.1:41081",
		"AuthorizationBearerToken": "UB42Gd0qUH6rkR4yxWbtTX85XCC9B0X1G7tFp64q9UlBjVdjZrtqaBIxFzj4dQvSiRYmxfF4hMAel6bw3xO7jnRgCGQBwBnjpPEfW1lrVAZFP3p55KxBra3mQDGrntE0",
		"TimeoutMilliseconds": 15000
	},

	"PolicyProvider": {
		"Type": "static_file",
		"Path": "policy.json"
	},

	"Misc": {
		"Debug": true
	}
}
```

## Fields

The configuration contains the following fields:

- `Matrix` - Matrix Homeserver and related configuration

	- `HomeserverDomainName` - the base domain name of your Matrix homeserver. This is what user identifiers contain (`@user:example.com`), and not necessarily the domain name where the Matrix homeserver is hosted (could actually be `matrix.example.com`)

	- `HomeserverApiEndpoint` - a URI to the Matrix homeserver's API. This would normally be a local address, as it's convenient to run `matrix-corporal` on the same machine as Matrix Synapse.

	- `AuthSharedSecret` - a shared secret between `matrix-corporal` and the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider Synapse module that you need to set up. You can generate it with something like: `pwgen -s 128 1`

	- `RegistrationSharedSecret` - the secret for Matrix Synapse's `/admin/register` API. Can be found in Matrix Synapse's `homeserver.yaml` file under the configuration key: `registration_shared_secret`

	- `TimeoutMilliseconds` - how long (in milliseconds) HTTP requests (from `matrix-corporal` to Matrix Synapse) are allowed to take before being timed out. Since clients often use long-polling for `/sync` (usually with a 30-second limit), setting this to a value of more than `30000` is recommended.

- `Corporal` - corporal-related configuration

	- `UserId` - a full Matrix user id of the system (needs to have admin privileges), which will be used to perform reconciliation and other tasks. This user account, with its admin privileges, will be used to find what users are available on the server, what their current state is, etc. This user account will also invite and kick users out of communities and rooms, so you need to make sure this user is joined to, and has the appropriate privileges, in all rooms and communities that you would like to manage.


- `Reconciliation` - reconciliation-related configuration

	- `RetryIntervalMilliseconds` - how long (in milliseconds) to wait before retrying reconciliation, in case the previous reconciliation attempt failed (due to Matrix Synapse being down, etc.).


- `HttpGateway` - [HTTP Gateway](http-gateway.md)-related configuration

	- `ListenAddress` - the network address to listen on. It's most likely a local one, as there's usually a reverse proxy (like nginx) capturing all traffic first and forwarding it here later on. If you're running this inside a container, use something like `0.0.0.0:41080`.

	- `TimeoutMilliseconds` - how long (in milliseconds) HTTP requests are allowed to take before being timed out. Since clients often use long-polling for `/sync` (usually with a 30-second limit), setting this to a value of more than `30000` is recommended. For this same reason, making this value larger than `Matrix.TimeoutMilliseconds` is required.

	- `InternalRESTAuth` - controls whether matrix-corporal's HTTP gateway will expose a `POST /_matrix/corporal/_matrix-internal/identity/v1/check_credentials` route, which can be used as a backend for [matrix-synapse-rest-password-provider](https://github.com/ma1uta/matrix-synapse-rest-password-provider). Enabling this is useful for making Interactive Authentication work. [Regular user authentication](user-authentication.md) works even without this, but during Interactive Auth, it's the homeserver that needs to contact us.
		- `Enabled` - whether this feature is enabled or not

		- `IPNetworkWhitelist` - an optional list of network ranges (e.g. `1.1.1.1/24`) that are allowed to access this authentication API. We don't rate-limit it (yet), so exposing it to every IP address is not a good idea.  If you define this as an empty list, all IP addresses are allowed. If you don't define this at all (or define it as `null`), we default to local/private IP ranges only.

	- `UserMappingResolver` - controls how `matrix-corporal` resolves access tokens for incoming requests to user IDs (internally, it uses the `/account/whoami` Client-Server API endpoint)
		- `CacheSize` (default: `10000`) - specifies the number of items that will be cached

		- `ExpirationTimeMilliseconds` (default `300000` = 5 minutes) - specifies how long before a cached item expires. After this time, the same incoming access token will have to be re-resolved by hitting the homeserver again. This can be important for [event hooks](event-hooks.md), if you rely on a hook's `meta.authenticatedMatrixUserID` data.


- `HttpApi` - HTTP API-related configuration

	- `Enabled` - whether the [HTTP API](http-api.md) is enabled or not

	- `ListenAddress` - the network address to listen on. It's most likely a local one, as there's usually a reverse proxy (like nginx) capturing all traffic first and forwarding it here later on. If you're running this inside a container, use something like `0.0.0.0:41081`.

	- `AuthorizationBearerToken` - a shared secret between `matrix-corporal` and your other remote system that will use its API. You can generate it with something like: `pwgen -s 128 1`

	- `TimeoutMilliseconds` - how long (in milliseconds) HTTP requests are allowed to take before being timed out.


- `PolicyProvider` - [policy provider](policy-providers.md) configuration.


- `Misc` - miscellaneous configuration

	- `Debug` - whether to enable debug mode or not (enable for more verbose logs)
