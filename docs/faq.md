# FAQ

## What is Matrix Corporal?

Matrix Corporal is a program, which sits in front of your [Matrix Synapse](https://github.com/matrix-org/synapse) homeserver.

It changes Matrix Synapse's configuration to make it match a [policy](policy.md) that you provide.

It also acts as a gateway to Matrix Synapse (sits in front of it), so that it can allow/deny requests according to the policy.


## When do I need Matrix Corporal?

You need Matrix Corporal when you want better management of your Matrix chat server.

That would be when you're in a corporate (or other similar team environment), where you would want automatic management of your users and their membership in rooms.

Using Matrix Corporal is a way to say:

> - these are my users that can use Matrix as of this moment
> - this is how they authenticate
> - these users need to be in this room
> - these other users need to be in that other room.

.. and have Matrix Corporal continually set up your Matrix server accordingly.


## Can I have users on my homeserver which are not managed by Matrix Corporal?

Yes. Matrix Corporal only manages the users that are part of the `users` field in the [policy](policy.md).

If there's no user policy configuration for a user in the policy's `users` field, that user is left untouched by Matrix Corporal. Such users can be managed by yourself, manually.


## Can I avoid forwarding all HTTP traffic to Matrix Corporal?

The short answer is: no.

The long answer is: it depends.

Matrix Corporal cannot do a good job of ensuring that everything works according to the [policy](policy.md), unless it can intercept and deny API requests that attempt to configure things differently.

Additionally, Matrix Corporal requires capturing the login API to make [user authentication](user-authentication.md) work.

It might be possible to forward traffic selectively (just the "dangerous" or otherwise-necessary routes), but those might change in future matrix-corporal/Synapse versions, so it's not recommended doing.


## Can users join other rooms?

Users are automatically joined and made to leave rooms according to the `joinedRoomIds` field in their [user policy](policy.md#user-policy-fields) and to the global `managedRoomIds` [policy field](policy.md#fields).

Users can create and join any room which is not listed in the global `managedRoomIds` policy field.

It's just rooms listed in the `managedRoomIds` that `matrix-corporal` cares about and controls tightly (requiring an explicit join rule in the `joinedRoomIds` list for that user).


## Can users join other communities?

Users are automatically joined and made to leave communities according to the `joinedCommunityIds` field in their [user policy](policy.md#user-policy-fields) and to the global `managedCommunityIds` [policy field](policy.md#fields).

Users can join any community which is not listed in the global `managedCommunityIds` policy field.

It's just communities listed in the `managedCommunityIds` that `matrix-corporal` cares about and controls tightly (requiring an explicit join rule in the `joinedCommunityIds` list for that user).


## Can Matrix Corporal be made to create rooms and communities?

Not for now.

You are responsible for creating all rooms and communities.
Once created through the reconciliator/system user (see `Reconciliator.UserId` in [configuration](configuration.md)), you define membership information in the [policy](policy.md) and users are made to join/leave accordingly.

Creating rooms and communities declaratively may be implemented in the future.


## Does Matrix Corporal require access to Matrix Synapse's database?

Not for now.
Matrix Corporal tries to do all its work by utilizing the Matrix server's client API.

Certain future features may require more tight coupling (and thus, database access), but there are no such features yet.


## Can I make Matrix Corporal add email addresses and other such information to user's profiles?

Not for now.

Attaching an email address to a user's profile cannot be done through the Matrix Client API, so Matrix Corporal also cannot do it.

It can be done if Matrix Corporal gets direct access to Matrix Synapse's database, but such integration hasn't been done yet. It may be done in the future.

Alternatively, the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider module may be modified, so that it would be able to do something similar, but such an implementation is deemed unclean at the moment.


## Can I use Matrix Corporal with homeservers other than Matrix Synapse?

Not for now.

Not that there are any other feature-complete homeservers at the time of writing this.. but..

Matrix Corporal tries to abstract away how it does things, so that other "connectors" can be implemented.
It tries to do mostly everything through the specced Matrix API, so it can theoretically be made to work with other homeservers.

However, it does use a few Synapse-specific APIs (`/admin/register` and other `/admin` APIs), as well as a Synapse-specific password provider in the form of [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth).
