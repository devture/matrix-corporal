package matrix

// ApiAdminEntityUser represents a user entity that is part of the list response
// at: GET /_matrix/client/r0/admin/users/{userId}
type ApiAdminEntityUser struct {
	Id           string `json:"name"`
	Admin        int    `json:"admin"`    //0 or 1
	Guest        int    `json:"is_guest"` //0 or 1
	PasswordHash string `json:"password_hash"`
}

// ApiWhoAmIResponse is a response as found at: GET /_matrix/client/r0/account/whoami
type ApiWhoAmIResponse struct {
	UserId string `json:"user_id"`
}

// ApiUserProfileResponse is a response as found at: GET /_matrix/client/r0/profile/{userId}
type ApiUserProfileResponse struct {
	AvatarUrl   string `json:"avatar_url"`
	DisplayName string `json:"displayname"`
}

// ApiUserProfileDisplayNameRequestPayload is a request payload for: POST /_matrix/client/r0/profile/{userId}/displayname
type ApiUserProfileDisplayNameRequestPayload struct {
	DisplayName string `json:"displayname"`
}

// ApiJoinedGroupsResponse is a response as found at: GET /_matrix/client/r0/joined_groups
type ApiJoinedGroupsResponse struct {
	GroupIds []string `json:"groups"`
}

// ApiAdminRegisterNonceResponse is a response as found at: GET /_matrix/client/r0/admin/register
type ApiUserAccountRegisterNonceResponse struct {
	Nonce string `json:"nonce"`
}

// ApiUserAccountRegisterRequestPayload is a request payload for: POST /_matrix/client/r0/admin/register
type ApiUserAccountRegisterRequestPayload struct {
	Nonce    string `json:"nonce"`
	Username string `json:"username"`
	Password string `json:"password"`
	Mac      string `json:"mac"`
	Type     string `json:"type"`
	Admin    bool   `json:"admin"`
}

// ApiUserAccountRegisterResponse is a response as found at: POST /_matrix/client/r0/admin/register
type ApiUserAccountRegisterResponse struct {
	AccessToken string `json:"access_token"`
	HomeServer  string `json:"home_server"`
	UserId      string `json:"user_id"`
}

// ApiCommunityInviteResponse is a response as found at: POST /_matrix/client/r0/groups/{communityId}/admin/users/invite/<invitee-id>
type ApiCommunityInviteResponse struct {
	State string `json:"state"`
}

// ApiCommunityInvitedUsersResponse is a response as found at: GET /_matrix/client/r0/groups/{communityId}/invited_users
type ApiCommunityInvitedUsersResponse struct {
	Chunk                  []ApiEntityCommunityInvitedUser `json:"chunk"`
	TotalUserCountEstimate int                             `json:"total_user_count_estimate"`
}

type ApiEntityCommunityInvitedUser struct {
	Id          string  `json:"user_id"`
	DisplayName string  `json:"displayname"`
	AvatarUrl   *string `json:"avatar_url"`
}
