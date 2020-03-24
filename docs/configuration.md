# Matrix Corporal Configuration

The `matrix-corporal` configuration is a JSON document that looks like this:

```json
{
	"Matrix": {
		"HomeserverDomainName": "matrix-corporal.127.0.0.1.xip.io",
		"HomeserverApiEndpoint": "http://matrix-corporal.127.0.0.1.xip.io:41408",
		"AuthSharedSecret": "7DXvheK1400ydCHjAymDU50FkeUedQJ2AYpitr3inLpSBIdRJN4kfS5IkGYvUptF",
		"RegistrationSharedSecret": "y4aTYam;zxKZ#MnaHRrGDPs4&dS*3VEv_&Ck_;pe1=CrtM8*=7",
		"TimeoutMilliseconds": 45000
	},

	"Reconciliation": {
		"UserId": "@matrix-corporal:matrix-corporal.127.0.0.1.xip.io",
		"RetryIntervalMilliseconds": 30000
	},

	"HttpGateway": {
		"ListenAddress": "127.0.0.1:41080"
	},

	"HttpApi": {
		"Enabled": true,
		"ListenAddress": "127.0.0.1:41081",
		"AuthorizationBearerToken": "UB42Gd0qUH6rkR4yxWbtTX85XCC9B0X1G7tFp64q9UlBjVdjZrtqaBIxFzj4dQvSiRYmxfF4hMAel6bw3xO7jnRgCGQBwBnjpPEfW1lrVAZFP3p55KxBra3mQDGrntE0"
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

	- `TimeoutMilliseconds` - how long (in milliseconds) HTTP requests (from `matrix-corporal` to Matrix Synapse) are allowed to take before being timed out. Since clients often use long-polling (usually with a 30-second limit), setting this to a value of more than `30000` is recommended. This value is also used to configure the timeout for `matrix-corporal`'s HTTP API and HTTP Gateway servers.


- `Reconciliation` - reconciliation-related configuration

	- `UserId` - a full Matrix user id of the system (needs to have admin privileges), which will be used to perform reconciliation. This user account, with its admin privileges, will be used to find what users are available on the server, what their current state is, etc. This user account will also invite and kick users out of communities and rooms, so you need to make sure this user is joined to, and has the appropriate privileges, in all rooms and communities that you would like to manage.

	- `RetryIntervalMilliseconds` - how long (in milliseconds) to wait before retrying reconciliation, in case the previous reconciliation attempt failed (due to Matrix Synapse being down, etc.).


- `HttpGateway` - [HTTP Gateway](http-gateway.md)-related configuration

	- `ListenAddress` - the network address to listen on. It's most likely a local one, as there's usually a reverse proxy (like nginx) capturing all traffic first and forwarding it here later on. If you're running this inside a container, use something like `0.0.0.0:41080`.


- `HttpApi` - HTTP API-related configuration

	- `Enabled` - whether the [HTTP API](http-api.md) is enabled or not

	- `ListenAddress` - the network address to listen on. It's most likely a local one, as there's usually a reverse proxy (like nginx) capturing all traffic first and forwarding it here later on. If you're running this inside a container, use something like `0.0.0.0:41081`.

	- `AuthorizationBearerToken` - a shared secret between `matrix-corporal` and your other remote system that will use its API. You can generate it with something like: `pwgen -s 128 1`


- `PolicyProvider` - [policy provider](policy-providers.md) configuration.


- `Misc` - miscellaneous configuration

	- `Debug` - whether to enable debug mode or not (enable for more verbose logs)
