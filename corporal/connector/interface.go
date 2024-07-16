package connector

import (
	"devture-matrix-corporal/corporal/avatar"
	"devture-matrix-corporal/corporal/matrix"
	"time"
)

type MatrixConnector interface {
	ObtainNewAccessTokenForUserId(userId, deviceId string, validUntil *time.Time) (string, error)
	VerifyAccessToken(userId, accessToken string) error
	DestroyAccessToken(userId, accessToken string) error
	LogoutAllAccessTokensForUser(ctx *AccessTokenContext, userId string) error

	DetermineCurrentState(ctx *AccessTokenContext, managedUserIds []string, adminUserId string) (*CurrentState, error)

	EnsureUserAccountExists(userId, password string) error

	GetUserProfileByUserId(ctx *AccessTokenContext, userId string) (*matrix.ApiUserProfileResponse, error)
	SetUserDisplayName(ctx *AccessTokenContext, userId string, displayName string) error
	SetUserAvatar(ctx *AccessTokenContext, userId string, avatar *avatar.Avatar) error

	InviteUserToRoom(ctx *AccessTokenContext, inviterId string, inviteeId string, roomId string) error
	UpdateRoomUserPowerLevel(ctx *AccessTokenContext, updaterId string, roomPowerForUserId map[string]int, roomId string) error
	JoinRoom(ctx *AccessTokenContext, userId string, roomId string) error
	LeaveRoom(ctx *AccessTokenContext, userId string, roomId string) error
}
