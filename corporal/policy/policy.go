package policy

import (
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/userauth"
	"fmt"
)

type Policy struct {
	SchemaVerson int `json:"schemaVersion"`

	// IdentificationStamp holds a policy identification value.
	// Policy providers/generators can attach any string value to a policy to help identify it.
	//
	// This could be a semver-like version, a timestamp, etc.
	// We don't care and we don't do anything with it for now.
	// In the future, we might suppress reconciliation if a new policy arrives and its identification stamp
	// matches the previous one.
	IdentificationStamp *string `json:"identificationStamp"`

	Flags PolicyFlags `json:"flags"`

	Hooks []*hook.Hook `json:"hooks"`

	ManagedCommunityIds []string `json:"managedCommunityIds"`

	ManagedRoomIds []string `json:"managedRoomIds"`

	User []*UserPolicy `json:"users"`
}

func (me *Policy) GetManagedUserIds() []string {
	var userIds []string
	for _, userPolicy := range me.User {
		userIds = append(userIds, userPolicy.Id)
	}
	return userIds
}

func (me *Policy) GetUserPolicyByUserId(userId string) *UserPolicy {
	for _, userPolicy := range me.User {
		if userPolicy.Id == userId {
			return userPolicy
		}
	}
	return nil
}

type PolicyFlags struct {
	// AllowCustomUserDisplayNames tells whether users are allowed to have display names,
	// which deviate from the ones in the policy.
	AllowCustomUserDisplayNames bool `json:"allowCustomUserDisplayNames"`

	// AllowCustomUserAvatars tells whether users are allowed to have avatars,
	// which deviate from the ones in the policy.
	AllowCustomUserAvatars bool `json:"allowCustomUserAvatars"`

	// AllowCustomPassthroughUserPasswords tells if managed users of AuthType=UserAuthTypePassthrough can change their password.
	// This is possible, because their password is stored and managed on the actual homeserver.
	// We can let password-changing requests go through.
	//
	// Users with another AuthType cannot change their password, because authentication happens on our side,
	// against the AuthCredential specified in the user's policy.
	AllowCustomPassthroughUserPasswords bool `json:"allowCustomPassthroughUserPasswords"`

	// AllowUnauthenticatedPasswordResets tells if unauthenticated users (no access token) can reset their password using the `/account/password` API.
	// They prove their identity by verifying 3pids before sending the unauthenticated request.
	// Corporal doesn't reach into the request data and can't figure out who it is, whether it's a policy-managed user, etc.,
	// so should you enable this option, Corporal can't effectively prevent password-changing for managed users.
	AllowUnauthenticatedPasswordResets bool `json:"allowUnauthenticatedPasswordResets"`

	// ForbidRoomCreation tells whether users are forbidden from creating rooms.
	// When there's a dedicated `UserPolicy` for the user, that one takes precedence over this default.
	ForbidRoomCreation bool `json:"forbidRoomCreation"`

	// ForbidEncryptedRoomCreation tells whether users are forbidden from creating encrypted rooms, and from switching rooms from unencrypted to encrypted.
	// When there's a dedicated `UserPolicy` for the user, that one takes precedence over this default.
	ForbidEncryptedRoomCreation bool `json:"forbidEncryptedRoomCreation"`

	// ForbidUnencryptedRoomCreation tells whether users are forbidden from creating unencrypted rooms.
	// When there's a dedicated `UserPolicy` for the user, that one takes precedence over this default.
	ForbidUnencryptedRoomCreation bool `json:"forbidUnencryptedRoomCreation"`

	// Allow3pidLogin tells whether login requests using an email address or phone number will be allowed to go through unmodified.
	// Enabling this may have security implications.
	// With this setting enabled, you're completely skipping matrix-corporal's login checks (`active` flag in the user policy, etc).
	Allow3pidLogin bool `json:"allow3pidLogin"`
}

type UserPolicy struct {
	Id     string `json:"id"`
	Active bool   `json:"active"`

	// AuthType's value is supposed to be one the `UserAuthType*` constants
	AuthType string `json:"authType"`

	// AuthCredential holds the up-to-date password for all auth types (other than UserAuthTypePassthrough).
	//
	// If AuthType is NOT UserAuthTypePassthrough, we intercept login requests and authenticate users against this value.
	//
	// If AuthType is UserAuthTypePassthrough, AuthCredential only serves as the initial password for the user account.
	// In such cases, authentication is performed by the homeserver, not by us.
	// Subsequent changes to AuthCredential (after the user account has been created) are not reflected.
	AuthCredential string `json:"authCredential"`

	DisplayName string `json:"displayName"`
	AvatarUri   string `json:"avatarUri"`

	JoinedCommunityIds []string `json:"joinedCommunityIds"`
	JoinedRoomIds      []string `json:"joinedRoomIds"`

	// ForbidRoomCreation tells whether this user is forbidden from creating rooms.
	ForbidRoomCreation *bool `json:"forbidRoomCreation"`

	// ForbidEncryptedRoomCreation tells whether this user is forbidden from creating encrypted rooms, and from switching rooms from unencrypted to encrypted.
	ForbidEncryptedRoomCreation *bool `json:"forbidEncryptedRoomCreation"`

	// ForbidUnencryptedRoomCreation tells whether this user is forbidden from creating unencrypted rooms.
	ForbidUnencryptedRoomCreation *bool `json:"forbidUnencryptedRoomCreation"`
}

func (me UserPolicy) Validate() error {
	if me.Id == "" {
		return fmt.Errorf("User has no id")
	}

	if !userauth.IsKnownUserAuthType(me.AuthType) {
		return fmt.Errorf("`%s` is an invalid auth type", me.AuthType)
	}

	return nil
}
