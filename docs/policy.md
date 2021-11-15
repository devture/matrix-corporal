# Policy

`matrix-corporal` acts upon a so-called policy - a JSON document telling it what the state of the Matrix server should be like.

Policies are loaded into `matrix-corporal` through the help of a [policy provider](policy-providers.md).


## Example

The policy is a JSON document that looks like this:

```json
{
	"schemaVersion": 1,

	"identificationStamp": null,

	"flags": {
		"allowCustomUserDisplayNames": false,
		"allowCustomUserAvatars": false,
		"allowCustomPassthroughUserPasswords": false,
		"forbidRoomCreation": false,
		"forbidEncryptedRoomCreation": false,
		"forbidUnencryptedRoomCreation": false
	},

	"managedCommunityIds": [
		"+a:example.com",
		"+b:example.com"
	],

	"managedRoomIds": [
		"!roomA:example.com",
		"!roomB:example.com",
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
			"eventType": "beforeAuthenticatedRequest",
			"routeMatchesRegex": "^/_matrix/client/r0/createRoom",
			"action": "consult.RESTServiceURL",
			"RESTServiceURL": "http://hook-rest-service:8080/reject/with-33-percent-chance",
			"RESTServiceRequestHeaders": {
				"Authorization": "Bearer SOME_TOKEN"
			},
			"RESTServiceContingencyHook": {
				"action": "reject",
				"responseStatusCode": 403,
				"rejectionErrorCode": "M_FORBIDDEN",
				"rejectionErrorMessage": "REST service down. Rejecting you to be on the safe side"
			}
		},

		{
			"id": "custom-hook-to-capture-room-creation-details",
			"eventType": "afterAuthenticatedRequest",
			"routeMatchesRegex": "^/_matrix/client/r0/createRoom",
			"action": "consult.RESTServiceURL",
			"RESTServiceURL": "http://hook-rest-service:8080/dump",
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
			"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"],
			"forbidRoomCreation": true
		},
		{
			"id": "@peter:example.com",
			"active": true,
			"authType": "sha1",
			"authCredential": "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			"displayName": "Just Peter",
			"avatarUri": "",
			"joinedCommunityIds": ["+b:example.com"],
			"joinedRoomIds": ["!roomB:example.com"],
			"forbidRoomCreation": false,
			"forbidEncryptedRoomCreation": true
		},
		{
			"id": "@george:example.com",
			"active": true,
			"authType": "rest",
			"authCredential": "https://intranet.example.com/_matrix-internal/identity/v1/check_credentials",
			"displayName": "Georgey",
			"avatarUri": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==",
			"joinedCommunityIds": ["+a:example.com", "+b:example.com"],
			"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"],
			"forbidRoomCreation": false,
			"forbidUnencryptedRoomCreation": true
		}
	]
}
```


## Fields

A policy contains the following fields:

- `schemaVersion` - tells which schema version this policy is using. This field will be useful in case we introduce backward-incompatible changes in the future. For now, it's always set to `1`.

- `identificationStamp` - an optional `string` value provided by you to help you identify this policy. For now, it's only used for debugging purposes, but in the future we might suppress reconciliation if we fetch a policy which has the same stamp as the one last used for reconciliation. So, if you provide this value at all, make sure it gets a new value, at least whenever the policy changes.

- `flags` - a list of flags telling `matrix-corporal` what other global restrictions to apply. See [flags](#flags) below.

- `managedCommunityIds` - a list of community identifiers (like `+community:server`) that `matrix-corporal` is allowed to manage for `users`. Any community that is not listed here will be left untouched.

- `managedRoomIds` - a list of room identifiers (like `!room:server`) that `matrix-corporal` is allowed to manage for `users`. Any room that is not listed here will be left untouched.

- `hooks` - a list of [event hooks](event-hooks.md) and their configuration.

- `users` - a list of users and their configuration (see [user policy fields](#user-policy-fields) below). Any server user that is not listed here will be left untouched.


## Flags

The following policy flags are supported:

- `allowCustomUserDisplayNames` (`true` or `false`, defaults to `false`) - controls whether users are allowed to set custom display names. By default, users are created with the display name specified in the policy. Whether they're able to set a custom one by themselves later on is controlled by this flag.

- `allowCustomUserAvatars` (`true` or `false`, defaults to `false`) - controls whether users are allowed to set custom avatar images. By default, users are created with the avatar image specified in the policy. Whether they're able to set a custom one by themselves later on is controlled by this flag.

- `allowCustomPassthroughUserPasswords` (`true` or `false`, defaults to `false`) - controls whether users with `authType=passthrough` can set custom passwords. By default, such users are created with an initial password as defined in `authCredential`. Whether they can change their homeserver password later or not is controlled by this flag.

- `allowUnauthenticatedPasswordResets` (`true` or `false`, defaults to `false`) - controls whether unauthenticated users (no access token) can reset their password using the `/_matrix/client/r0/account/password` API. They prove their identity by verifying 3pids before sending the unauthenticated request. `matrix-corporal` doesn't reach into the `auth` request data for this endpoint and can't figure out who it is and whether it's a policy-managed user or not and what policy it should apply. Should you enable this option, all users will be allowed to reset their Synapse-stored password. If all your users are managed by `matrix-corporal` and have passwords in its policy, you'd better not enable this.

- `forbidRoomCreation` (`true` or `false`, defaults to `false`) - controls whether users are forbidden from creating rooms. The `forbidRoomCreation` [User policy field](#user-policy-fields) takes precedence over this. This is just a global default in case the user policy does not specify a value.

- `forbidEncryptedRoomCreation` (`true` or `false`, defaults to `false`) - controls whether users are forbidden from creating encrypted rooms and from switching unencrypted rooms to encrypted subsequently. The `forbidEncryptedRoomCreation` [User policy field](#user-policy-fields) takes precedence over this. This is just a global default in case the user policy does not specify a value. Also, see the [note about encryption](#notes-about-controlling-room-encryption) below.

- `forbidUnencryptedRoomCreation` (`true` or `false`, defaults to `false`) - controls whether users are forbidden from creating unencrypted rooms. The `forbidUnencryptedRoomCreation` [User policy field](#user-policy-fields) takes precedence over this. This is just a global default in case the user policy does not specify a value. Also, see the [note about encryption](#notes-about-controlling-room-encryption) below.

- `allow3pidLogin` (`true` or `false`, defaults to `false`) - controls whether users would be able to log in with 3pid (third-party identifiers) associated with their user account (email address / phone number). If enabled, we let such login requests requests pass and go directly to the homeserver. This has some security implications - any checks matrix-corporal would have normally done (checking the `active` status in the user policy, etc.) are skipped.

## User policy fields

The `users` field in the [policy fields](#fields) (above) contains a list of users and the configuration that applies to each user (besides the global [policy flags](#flags)).

A user policy object looks like this:

```json
{
	"id": "@john:example.com",
	"active": true,
	"authType": "plain",
	"authCredential": "PaSSw0rD",
	"displayName": "John",
	"avatarUri": "https://example.com/john.jpg",
	"joinedCommunityIds": ["+a:example.com"],
	"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"],
	"forbidRoomCreation": false,
	"forbidEncryptedRoomCreation": false,
	"forbidUnencryptedRoomCreation": false
}
```


A user-policy contains the following fields:

- `id` - the full Matrix id of the user

- `active` (`true` or `false`) - tells whether the user's account is active. If `false`: the account will not be created on the Matrix server or it will be disabled, if it exists. Access to disabled accounts is revoked immediately (destroying access tokens).

- `authType` - the type of authentication to use for this user. See [User Authentication](user-authentication.md) for more information.

- `authCredential` - the authentication credential to use for this user. This has a different meaning depending on the type of authenticator being used (specified in the `authType` field). See [User Authentication](user-authentication.md) for more information.

- `displayName` - the name of this user. New accounts will always be created with the name specified in the policy. The display name on the Matrix server is kept in sync with the policy (and any edits by the user are prevented), unless the `allowCustomUserDisplayNames` flag is set to `true` (see [flags](#flags) above).

- `avatarUri` - the avatar image of this user. It can be a public remote URL or a [data URI](https://en.wikipedia.org/wiki/Data_URI_scheme) (e.g. `data:image/png;base64,DATA_GOES_HERE`). New accounts will always be created with the avatar specified in the policy. The avatar on the Matrix server is kept in sync with the policy (and any edits by the user are prevented), unless the `allowCustomUserAvatars` flag is set to `true` (see [flags](#flags) above). For performance reasons, avatar URLs are not re-fetched unless the URL changes, so make sure avatar URLs change when the underlying data changes.

- `joinedCommunityIds` - a list of community identifiers (e.g. `+community:server`) that the user is part of. The user will be auto-joined to any communities listed here, unless already joined. If the user happens to be joined to a community which is not listed here, but appears in the top-level `managedCommunityIds` field, the user will be kicked out of that community. The user can be part of any number of other communities which are not listed in `joinedCommunityIds`, as long as they are also not listed in `managedCommunityIds`.

- `joinedRoomIds` - a list of room identifiers (e.g. `!room:server`) that the user is part of. The user will be auto-joined to any rooms listed here, unless already joined. If the user happens to be joined to a room which is not listed here, but appears in the top-level `managedRoomIds` field, the user will be kicked out of that room. The user can be part of any number of other room which are not listed in `joinedRoomIds`, as long as they are also not listed in `managedRoomIds`.

- `forbidRoomCreation` (`true` or `false`, defaults to `false`) - controls whether this user is forbidden from creating rooms. If this field is omitted, the global `forbidRoomCreation` [flag](#flags) is used as a fallback.

- `forbidEncryptedRoomCreation` (`true` or `false`, defaults to `false`) - controls whether this user is forbidden from creating encrypted rooms and from switching unencrypted rooms to encrypted subsequently. If this field is omitted, the global `forbidEncryptedRoomCreation` [flag](#flags) is used as a fallback. Also, see the [note about encryption](#notes-about-controlling-room-encryption) below.

- `forbidUnencryptedRoomCreation` (`true` or `false`, defaults to `false`) - controls whether this user is forbidden from creating unencrypted rooms. If this field is omitted, the global `forbidUnencryptedRoomCreation` [flag](#flags) is used as a fallback. Also, see the [note about encryption](#notes-about-controlling-room-encryption) below.


## Notes about controlling room encryption

We support `forbidEncryptedRoomCreation` and `forbidUnencryptedRoomCreation` flags both as a [global level flag](#flags) and as a [user policy flag](#user-policy-fields).

Preventing encrypted or unencrypted rooms from being created does not guarantee that users will not end up being part of such rooms. If your server is a federating one, your users may end up in rooms which don't respect these value.


## Generating the policy file

You can generate the matrix-corporal policy file directly (from your own software), or with the help of some other tool.

If you're using LDAP, you may find this tool useful: [cmuller/ldap_matrix](https://github.com/cmuller/ldap_matrix) - Matrix Corporal Policy Specification Using LDAP Groups
