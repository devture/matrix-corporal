# Policy

`matrix-corporal` acts upon a so-called policy - a JSON document telling it what the state of the Matrix server should be like.

Policies are loaded into `matrix-corporal` through the help of a [policy provider](policy-providers.md).


## Example

The policy is a JSON document that looks like this:

```yaml
{
	"schemaVersion": 1,

	"flags": {
		"allowCustomUserDisplayNames": false,
		"allowCustomUserAvatars": false
	},

	"managedCommunityIds": [
		"+a:example.com",
		"+b:example.com"
	],

	"managedRoomIds": [
		"!roomA:example.com",
		"!roomB:example.com",
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


## Fields

A policy contains the following fields:

- `schemaVersion` - tells which schema version this policy is using. This field will be useful in case we introduce backward-incompatible changes in the future. For now, it's always set to `1`.

- `flags` - a list of flags telling `matrix-corporal` what other global restrictions to apply. See [flags](#flags) below.

- `managedCommunityIds` - a list of community identifiers (like `+community:server`) that `matrix-corporal` is allowed to manage for `users`. Any community that is not listed here will be left untouched.

- `managedRoomIds` - a list of room identifiers (like `!room:server`) that `matrix-corporal` is allowed to manage for `users`. Any room that is not listed here will be left untouched.

- `users` - a list of users and their configuration (see [user policy fields](#user-policy-fields) below). Any server user that is not listed here will be left untouched.


## Flags

The following policy flags are supported:

- `allowCustomUserDisplayNames` (`true` or `false`, defaults to `false`) - controls whether users are allowed to set custom display names. By default, users are created with the display name specified in the policy. Whether they're able to set a custom one by themselves later on is controlled by this flag.

- `allowCustomUserAvatars` (`true` or `false`, defaults to `false`) - controls whether users are allowed to set custom avatar images. By default, users are created with the avatar image specified in the policy. Whether they're able to set a custom one by themselves later on is controlled by this flag.


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
	"joinedRoomIds": ["!roomA:example.com", "!roomB:example.com"]
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