package policy

import "devture-matrix-corporal/corporal/hook"

type Policy struct {
	SchemaVerson int `json:"schemaVersion"`

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

	// ForbidRoomCreation tells whether users are forbidden from creating rooms.
	// When there's a dedicated `UserPolicy` for the user, that one takes precedence over this default.
	ForbidRoomCreation bool `json:"forbidRoomCreation"`
}

const (
	UserAuthTypePassthrough = "passthrough"
	UserAuthTypeMd5         = "md5"
	UserAuthTypeSha1        = "sha1"
	UserAuthTypeSha256      = "sha256"
	UserAuthTypeSha512      = "sha512"
)

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

	// Tells whether this user is forbidden from creating rooms.
	ForbidRoomCreation *bool `json:"forbidRoomCreation"`
}
