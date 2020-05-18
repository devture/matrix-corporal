package policy

type Policy struct {
	SchemaVerson int `json:"schemaVersion"`

	Flags PolicyFlags `json:"flags"`

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
	// Tells whether users are allowed to have display names,
	// which deviate from the ones in the policy.
	AllowCustomUserDisplayNames bool `json:"allowCustomUserDisplayNames"`

	// Tells whether users are allowed to have avatars,
	// which deviate from the ones in the policy.
	AllowCustomUserAvatars bool `json:"allowCustomUserAvatars"`

	// Tells whether users are forbidden from creating rooms.
	// When there's a dedicated `UserPolicy` for the user, that one takes precedence over this default.
	ForbidRoomCreation bool `json:"forbidRoomCreation"`
}

type UserPolicy struct {
	Id     string `json:"id"`
	Active bool   `json:"active"`

	AuthType       string `json:"authType"`
	AuthCredential string `json:"authCredential"`

	DisplayName string `json:"displayName"`
	AvatarUri   string `json:"avatarUri"`

	JoinedCommunityIds []string `json:"joinedCommunityIds"`
	JoinedRoomIds      []string `json:"joinedRoomIds"`

	// Tells whether this user is forbidden from creating rooms.
	ForbidRoomCreation *bool `json:"forbidRoomCreation"`

	//PowerLevel.
	PowerLevel int `json:"powerLevel"`

}
