package connector

import (
	"devture-matrix-corporal/corporal/avatar"
	"devture-matrix-corporal/corporal/matrix"
)

type MatrixConnector interface {
	CreateAccessTokenContext(deviceId string) *AccessTokenContext

	ObtainNewAccessTokenForUserId(userId, deviceId string) (string, error)
	DestroyAccessToken(userId, accessToken string) error
	LogoutAllAccessTokensForUser(ctx *AccessTokenContext, userId string) error

	DetermineCurrentState(ctx *AccessTokenContext, managedUserIds []string, adminUserId string) (*CurrentState, error)

	EnsureUserAccountExists(userId, password string) error

	GetUserProfileByUserId(ctx *AccessTokenContext, userId string) (*matrix.ApiUserProfileResponse, error)
	SetUserDisplayName(ctx *AccessTokenContext, userId string, displayName string) error
	SetUserAvatar(ctx *AccessTokenContext, userId string, avatar *avatar.Avatar) error

	InviteUserToCommunity(ctx *AccessTokenContext, inviterId string, inviteeId string, communityId string) error
	AcceptCommunityInvite(ctx *AccessTokenContext, userId string, communityId string) error
	KickUserFromCommunity(ctx *AccessTokenContext, kickerUserId string, kickeeUserId string, communityId string) error

	InviteUserToRoom(ctx *AccessTokenContext, inviterId string, inviteeId string, roomId string) error
	JoinRoom(ctx *AccessTokenContext, userId string, roomId string) error
	LeaveRoom(ctx *AccessTokenContext, userId string, roomId string) error
}
