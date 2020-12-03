[![Support room on Matrix](https://img.shields.io/matrix/matrix-corporal:devture.com.svg?label=%23matrix-corporal%3Adevture.com&logo=matrix&style=for-the-badge&server_fqdn=matrix.devture.com)](https://matrix.to/#/#matrix-corporal:devture.com) [![donate](https://liberapay.com/assets/widgets/donate.svg)](https://liberapay.com/s.pantaleev/donate)

# Matrix Corporal: reconciliator and gateway for a managed Matrix server

`matrix-corporal` manages your [Matrix](http://matrix.org/) server according to a configuration policy.

The point is to have a single source of truth about users/rooms/communities somewhere
(say in an external system, like your intranet),
and have something (`matrix-corporal`) continually reconfigure your Matrix server in accordance with it.

In a way, it can be thought of as "Kubernetes for Matrix", in that it takes such a JSON policy as an input,
and performs **reconciliation** with the Matrix server -- creating, activating, disabling user accounts, making them (automatically) join/leave rooms and communities, etc.

Besides reconciliation, `matrix-policy` also does **firewalling** (acts as a gateway).
You can put `matrix-corporal` in front of your [Matrix Synapse](https://github.com/matrix-org/synapse) server,
and have it capture all Matrix API requests and allow/deny them in accordance with the policy.

With **reconciliation** and **firewalling** both working together, `matrix-corporal` ensures
that your Matrix server's state always matches what the policy says, and that no user
is allowed to perform actions which take the server out of that equilibrium.

For more information, read below or jump to the [FAQ](docs/faq.md).


## Features

You give `matrix-corporal` a [policy](docs/policy.md) document by some means (some [policy provider](docs/policy-providers.md), and it takes care of the following things for you:

- creating user accounts according to the [policy](docs/policy.md) or disabling user accounts and revoking access

- authenticating users according to the policy (plain-text passwords, hashed passwords, REST auth)

- changing user profile data (names and avatars), to keep them in sync with the policy

- changing user room/community memberships, to keep them in sync with the policy

- allowing or denying Matrix API requests, to prevent the server state deviating from the policy


## Example

It's probably best explained with an example. Here's a [policy](docs/policy.md) that `matrix-corporal` can work with:

```json
{
	"schemaVersion": 1,

	"flags": {
		"allowCustomUserDisplayNames": false,
		"allowCustomUserAvatars": false
	},

	"managedRoomIds": [
		"!roomA:example.com",
		"!roomB:example.com",
	],

	"managedCommunityIds": [
		"+a:example.com",
		"+b:example.com"
	],

	"hooks": [
		{
			"id": "custom-hook-to-prevent-banning",
			"eventType": "beforeAnyRequest",
			"routeMatchesRegex": "^/_matrix/client/r0/rooms/([^/]+)/ban",
			"methodMatchesRegex": "POST",
			"action": "reject",
			"responseStatusCode": 403,
			"rejectionErrorCode": "M_FORBIDDEN",
			"rejectionErrorMessage": "Banning is forbidden on this server. We're nice like that!"
		},

		{
			"id": "custom-hook-to-reject-room-creation-once-in-a-while",
			"eventType": "beforeAuthenticatedPolicyCheckedRequest",
			"routeMatchesRegex": "^/_matrix/client/r0/createRoom",
			"action": "consult.RESTServiceURL",
			"RESTServiceURL": "http://hook-rest-service:8080/reject/with-33-percent-chance",
			"RESTServiceRequestHeaders": {
				"Authorization": "Bearer SOME_TOKEN"
			}
		}
	],

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
		},
		{
			"id": "@peter:example.com",
			"active": true,
			"authType": "sha1",
			"authCredential": "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			"displayName": "Just Peter",
			"avatarUri": "",
			"joinedCommunityIds": ["+b:example.com"],
			"joinedRoomIds": ["!roomB:example.com"]
		},
		{
			"id": "@george:example.com",
			"active": true,
			"authType": "rest",
			"authCredential": "https://intranet.example.com/_matrix-internal/identity/v1/check_credentials",
			"displayName": "Georgey",
			"avatarUri": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==",
			"joinedCommunityIds": ["+a:example.com", "+b:example.com"],
			"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"]
		}
	]
}
```

The JSON [policy](docs/policy.md) above, describes the state that your server should have:

- managed communities - a list of communities that you want `matrix-corporal` to manage for you. Any other communities are untouched.

- managed rooms - a list of rooms that you want `matrix-corporal` to manage for you. Any other rooms are untouched.

- managed users (including their profile details and authentication data). Any other users are untouched.

- membership information (which users need to be in which communities/rooms). Any other memberships are untouched.


As a result, `matrix-corporal` will perform a sequence of actions, ensuring that:

- all users are created and that their corresponding credentials are made to work

- all user details are made to match the policy (names, avatars, etc.)

- inactive users will be disabled and prevented from logging in

- users are automatically joined to or kicked out of the specified communities and rooms

Any time you change the [policy](docs/policy.md) in the future, `matrix-corporal` acts upon the Matrix server,
so that its state is made to match the policy.


## Installation

To configure and install `matrix-corporal` on your own server, follow the [README in the docs/ directory](docs/README.md).


## Development / Experimenting

To give `matrix-corporal` a try (without actually installing it anywhere) or to do development on it, refer to the [development introduction](docs/development.md).


## Support

Matrix room: [#matrix-corporal:devture.com](https://matrix.to/#/#matrix-corporal:devture.com)

Github issues: [devture/matrix-corporal/issues](https://github.com/devture/matrix-corporal/issues)
