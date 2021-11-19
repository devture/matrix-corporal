package matrix

// ApiLoginRequestPayload represents is a request payload for: POST /_matrix/client/{apiVersion:(r0|v3)}/login
type ApiLoginRequestPayload struct {
	// Type is matrix.LoginTypeToken or something else
	Type string `json:"type"`

	Password string `json:"password,omitempty"`
	Medium   string `json:"medium,omitempty"`

	// User is deprecated in favor of Identifier
	User string `json:"user,omitempty"`

	Address                  string `json:"address,omitempty"`
	Token                    string `json:"token,omitempty"`
	DeviceID                 string `json:"device_id,omitempty"`
	InitialDeviceDisplayName string `json:"initial_device_display_name,omitempty"`

	Identifier ApiLoginRequestIdentifier `json:"identifier"`
}

type ApiLoginRequestIdentifier struct {
	// Type is matrix.LoginIdentifierTypeUser (`m.id.user`) or something else.
	// Different identifier types contain different fields.
	Type string `json:"type"`

	// User contains the username of the user logging in, when Type = matrix.LoginIdentifierTypeUser.
	User string `json:"user"`
}

// ApiAdminResponseUserLogin represents a login response payload
// at: POST /_synapse/admin/v1/users/<user_id>/login
type ApiAdminResponseUserLogin struct {
	AccessToken string `json:"access_token"`
}

// ApiAdminEntityUser represents a user entity that is part of the list response
// at: GET /_synapse/admin/v2/users
type ApiAdminResponseUsers struct {
	Users []ApiAdminEntityUser `json:"users"`
}

// ApiAdminEntityUser represents a user entity that is part of the list response
// at: GET /_synapse/admin/v2/users
type ApiAdminEntityUser struct {
	Id           string `json:"name"`
	Admin        int    `json:"admin"`    //0 or 1
	Guest        int    `json:"is_guest"` //0 or 1
	PasswordHash string `json:"password_hash"`
	DisplayName  string `json:"displayname"`
	AvatarURL    string `json:"avatar_url"`
}

// ApiWhoAmIResponse is a response as found at: GET /_matrix/client/{apiVersion:(r0|v3)}/account/whoami
type ApiWhoAmIResponse struct {
	UserId string `json:"user_id"`
}

// ApiUserProfileResponse is a response as found at: GET /_matrix/client/{apiVersion:(r0|v3)}/profile/{userId}
type ApiUserProfileResponse struct {
	AvatarUrl   string `json:"avatar_url"`
	DisplayName string `json:"displayname"`
}

// ApiUserProfileDisplayNameRequestPayload is a request payload for: POST /_matrix/client/{apiVersion:(r0|v3)}/profile/{userId}/displayname
type ApiUserProfileDisplayNameRequestPayload struct {
	DisplayName string `json:"displayname"`
}

// ApiJoinedGroupsResponse is a response as found at: GET /_matrix/client/{apiVersion:(r0|v3)}/joined_groups
type ApiJoinedGroupsResponse struct {
	GroupIds []string `json:"groups"`
}

// ApiAdminRegisterNonceResponse is a response as found at: GET /_matrix/client/{apiVersion:(r0|v3)}/admin/register
type ApiUserAccountRegisterNonceResponse struct {
	Nonce string `json:"nonce"`
}

// ApiUserAccountRegisterRequestPayload is a request payload for: POST /_matrix/client/{apiVersion:(r0|v3)}/admin/register
type ApiUserAccountRegisterRequestPayload struct {
	Nonce    string `json:"nonce"`
	Username string `json:"username"`
	Password string `json:"password"`
	Mac      string `json:"mac"`
	Type     string `json:"type"`
	Admin    bool   `json:"admin"`
}

// ApiUserAccountRegisterResponse is a response as found at: POST /_matrix/client/{apiVersion:(r0|v3)}/admin/register
type ApiUserAccountRegisterResponse struct {
	AccessToken string `json:"access_token"`
	HomeServer  string `json:"home_server"`
	UserId      string `json:"user_id"`
}

// ApiCommunityInviteResponse is a response as found at: POST /_matrix/client/{apiVersion:(r0|v3)}/groups/{communityId}/admin/users/invite/<invitee-id>
type ApiCommunityInviteResponse struct {
	State string `json:"state"`
}

// ApiCommunityInvitedUsersResponse is a response as found at: GET /_matrix/client/{apiVersion:(r0|v3)}/groups/{communityId}/invited_users
type ApiCommunityInvitedUsersResponse struct {
	Chunk                  []ApiEntityCommunityInvitedUser `json:"chunk"`
	TotalUserCountEstimate int                             `json:"total_user_count_estimate"`
}

type ApiEntityCommunityInvitedUser struct {
	Id          string  `json:"user_id"`
	DisplayName string  `json:"displayname"`
	AvatarUrl   *string `json:"avatar_url"`
}
