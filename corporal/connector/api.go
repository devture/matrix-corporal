package connector

import (
	"devture-matrix-corporal/corporal/avatar"
	"devture-matrix-corporal/corporal/matrix"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Jeffail/gabs"
	"github.com/matrix-org/gomatrix"
)

const (
	accountDataTypeAvatarSourceUriHashes = "com.devture.matrix.corporal.avatar_source_uri_hashes"
)

// ApiConnector is an abstract implementation of MatrixConnector for integrating with a Matrix server via API.
// Another connector (like SynapseConnector) would extend this with the remaining functionality
type ApiConnector struct {
	homeserverApiEndpoint             string
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator
	logger                            *logrus.Logger

	httpClient *http.Client
}

func NewApiConnector(
	homeserverApiEndpoint string,
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator,
	timeoutMilliseconds int,
	logger *logrus.Logger,
) *ApiConnector {
	// We've had certain versions of Synapse (like 0.33.2) get stuck forever while processing requests.
	// It's hard to debug when it happens, because we get stuck too.
	// We never want to get stuck, so we'll use our own http client for gomatrix (set in createMatrixClientForUserIdAndToken()).
	httpClient := &http.Client{
		Timeout: time.Duration(timeoutMilliseconds) * time.Millisecond,
	}

	return &ApiConnector{
		homeserverApiEndpoint:             homeserverApiEndpoint,
		sharedSecretAuthPasswordGenerator: sharedSecretAuthPasswordGenerator,
		logger:                            logger,

		httpClient: httpClient,
	}
}

func (me *ApiConnector) ObtainNewAccessTokenForUserId(userId, deviceId string, validUntil *time.Time) (string, error) {
	// We ignore validUntil, because the specced /login API does not support token expiration (yet).

	client, _ := me.createMatrixClientForUserIdAndToken("", "")

	var resp *gomatrix.RespLogin
	err := matrix.ExecuteWithRateLimitRetries(me.logger, "user.obtain_access_token", func() error {
		payload := &matrix.ApiLoginRequestPayload{
			Type: matrix.LoginTypePassword,

			// Old deprecated field
			User: userId,

			Identifier: matrix.ApiLoginRequestIdentifier{
				Type: matrix.LoginIdentifierTypeUser,
				User: userId,
			},

			Password: me.sharedSecretAuthPasswordGenerator.GenerateForUserId(userId),
			DeviceID: deviceId,
		}

		return client.MakeRequest("POST", client.BuildURL("/login"), payload, &resp)
	})

	if err != nil {
		return "", err
	}

	return resp.AccessToken, nil
}

func (me *ApiConnector) DestroyAccessToken(userId, accessToken string) error {
	client, _ := gomatrix.NewClient(me.homeserverApiEndpoint, userId, accessToken)
	_, err := client.Logout()

	if matrix.IsErrorWithCode(err, matrix.ErrorUnknownToken) {
		//Suppress errors for access tokens that appear to be non-working already.
		return nil
	}

	return err
}

func (me *ApiConnector) LogoutAllAccessTokensForUser(ctx *AccessTokenContext, userId string) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	err = client.MakeRequest("POST", client.BuildURL("/logout/all"), nil, nil)
	if err != nil {
		return err
	}

	// Doing "logout all" also logs out the token we've used just now.
	// To ensure our access token context can serve others later on,
	// let's remove this token from it. This way, it would obtain a new one later.
	ctx.ClearAccessTokenForUserId(userId)

	return nil
}

func (me *ApiConnector) DetermineCurrentState(
	ctx *AccessTokenContext,
	managedUserIds []string,
	adminUserId string,
) (*CurrentState, error) {
	// This cannot be implemented using standard (implementation-agnostic) Client-Server APIs.
	return nil, fmt.Errorf("Not implemented")
}

func (me *ApiConnector) getUserStateByUserId(
	ctx *AccessTokenContext,
	userId string,
) (*CurrentUserState, error) {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	joinedCommunityIds, err := me.getJoinedCommunityIdsByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	joinedRoomIds, err := me.getJoinedRoomIdsByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	userProfile, err := me.GetUserProfileByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	displayName := userProfile.DisplayName
	isDeactivated := matrix.IsUserDeactivatedAccordingToDisplayName(displayName)
	if isDeactivated {
		// Clean up the display name from the deactivation marker.
		// We don't want to give all other code the wrong impression,
		// potentially kicking off "display name set" actions, etc.
		displayName = matrix.CleanDeactivationMarkerFromDisplayName(displayName)
	}

	var avatarSourceUriHash string
	if userProfile.AvatarUrl == "" {
		// Not having an avatar is equivalent to deriving from an empty source avatar URI.
		// Let's build a hash like that.
		avatarSourceUriHash = avatar.UriHash("")
	} else {
		avatarSourceUriHash, err = me.determineAvatarSourceUriHashByUserAndMxcUri(
			ctx,
			userId,
			userProfile.AvatarUrl,
		)
		if err != nil {
			return nil, err
		}
	}

	return &CurrentUserState{
		Id:                  client.UserID,
		Active:              !isDeactivated,
		DisplayName:         displayName,
		AvatarMxcUri:        userProfile.AvatarUrl,
		AvatarSourceUriHash: avatarSourceUriHash,
		JoinedCommunityIds:  joinedCommunityIds,
		JoinedRoomIds:       joinedRoomIds,
	}, nil
}

func (me *ApiConnector) storeAvatarSourceUriHashForUserAndMxcUri(
	ctx *AccessTokenContext,
	userId string,
	mxcUri string,
	avatarSourceUriHash string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	payload := map[string]string{
		mxcUri: avatarSourceUriHash,
	}

	err = matrix.ExecuteWithRateLimitRetries(me.logger, "user.set_account_data", func() error {
		// We'll completely overwrite the old account data at that key,
		// storing only the avatar hash for the given mxcUri and purging everything else.
		return client.MakeRequest(
			"PUT",
			client.BuildURL(
				fmt.Sprintf(
					"/user/%s/account_data/%s",
					userId,
					accountDataTypeAvatarSourceUriHashes,
				),
			),
			payload,
			nil,
		)
	})

	return err
}

func (me *ApiConnector) determineAvatarSourceUriHashByUserAndMxcUri(
	ctx *AccessTokenContext,
	userId string,
	mxcUri string,
) (string, error) {
	accountDataPayload, err := me.GetUserAccountDataContentByType(
		ctx,
		userId,
		accountDataTypeAvatarSourceUriHashes,
	)
	if err != nil {
		return "", err
	}

	value, ok := accountDataPayload[mxcUri]
	if !ok {
		return "", nil
	}

	valueAsString, ok := value.(string)
	if !ok {
		return "", nil
	}

	return valueAsString, nil
}

func (me *ApiConnector) EnsureUserAccountExists(userId, password string) error {
	// This cannot be implemented using standard (implementation-agnostic) Client-Server APIs.
	return fmt.Errorf("Not implemented")
}

func (me *ApiConnector) GetUserAccountDataContentByType(
	ctx *AccessTokenContext,
	userId string,
	accountDataType string,
) (map[string]interface{}, error) {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	var accountData map[string]interface{}
	err = client.MakeRequest(
		"GET",
		client.BuildURL(
			fmt.Sprintf("/user/%s/account_data/%s", userId, accountDataType),
		),
		nil,
		&accountData,
	)

	if err != nil {
		if matrix.IsErrorWithCode(err, matrix.ErrorNotFound) {
			// No such account data
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	return accountData, nil
}

func (me *ApiConnector) getJoinedCommunityIdsByUserId(
	ctx *AccessTokenContext,
	userId string,
) ([]string, error) {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	var resp matrix.ApiJoinedGroupsResponse

	err = client.MakeRequest("GET", client.BuildURL("/joined_groups"), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.GroupIds, nil
}

func (me *ApiConnector) GetUserProfileByUserId(ctx *AccessTokenContext, userId string) (*matrix.ApiUserProfileResponse, error) {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	var resp matrix.ApiUserProfileResponse

	err = client.MakeRequest("GET", client.BuildURL(fmt.Sprintf("/profile/%s", client.UserID)), nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (me *ApiConnector) getJoinedRoomIdsByUserId(
	ctx *AccessTokenContext,
	userId string,
) ([]string, error) {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	resp, err := client.JoinedRooms()
	if err != nil {
		return nil, err
	}

	return resp.JoinedRooms, nil
}

func (me *ApiConnector) SetUserAvatar(
	ctx *AccessTokenContext,
	userId string,
	avatar *avatar.Avatar,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	if avatar.ContentType == "" {
		// This is a request for avatar removal.

		// We'll just deassociate the avatar from the user's profile, without deleting the actual image.
		// We don't know if the image is something we've put there or not,
		// and we don't know if it's used elsewhere.
		// It's not our job to delete it.

		return matrix.ExecuteWithRateLimitRetries(me.logger, "user.set_avatar", func() error {
			return client.SetAvatarURL("")
		})
	}

	// Request for setting a new avatar.

	// There may be an old avatar whose image we're leaving behind. We intentionally do not care.
	// We apply the same reasoning as above (for avatar removal).

	// This request cannot be retried so easily, as we'd need to rewind the Body somehow.
	resp, err := client.UploadToContentRepo(avatar.Body, avatar.ContentType, avatar.ContentLength)
	if err != nil {
		return fmt.Errorf("Failed uploading avatar: %s", err)
	}

	mxcUri := resp.ContentURI

	err = matrix.ExecuteWithRateLimitRetries(me.logger, "user.set_avatar", func() error {
		return client.SetAvatarURL(mxcUri)
	})
	if err != nil {
		return fmt.Errorf("Failed setting avatar: %s", err)
	}

	// To keep track of what this avatar is derived from, store a mapping
	// between the file and the uri hash of its source.
	err = matrix.ExecuteWithRateLimitRetries(me.logger, "user.store_avatar_source_uri_hash", func() error {
		return me.storeAvatarSourceUriHashForUserAndMxcUri(ctx, userId, mxcUri, avatar.UriHash)
	})
	if err != nil {
		return fmt.Errorf("Failed storing avatar URI to avatar source uri hash mapping: %s", err)
	}

	return nil
}

func (me *ApiConnector) SetUserDisplayName(
	ctx *AccessTokenContext,
	userId string,
	displayName string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "user.set_display_name", func() error {
		return client.SetDisplayName(displayName)
	})
}

func (me *ApiConnector) InviteUserToCommunity(
	ctx *AccessTokenContext,
	inviterId string,
	inviteeId string,
	communityId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, inviterId)
	if err != nil {
		return err
	}

	var response matrix.ApiCommunityInviteResponse

	err = matrix.ExecuteWithRateLimitRetries(me.logger, "community.invite", func() error {
		return client.MakeRequest(
			"PUT",
			client.BuildURL(fmt.Sprintf(
				"/groups/%s/admin/users/invite/%s",
				communityId,
				inviteeId,
			)),
			map[string]interface{}{},
			&response,
		)
	})
	if err == nil {
		return nil
	}

	// At the moment, Synapse would respond with 500 / M_UNKNOWN / Internal Server Error
	// if we try to invite the same user twice.
	// Below, we'll attempt to detect and ignore this error.
	// When [this](https://github.com/matrix-org/synapse/issues/3623) gets fixed, it will be simpler.

	if !matrix.IsErrorWithCode(err, matrix.ErrorUnknown) {
		return err
	}

	isInvited, errInv := me.isUserIdInvitedToCommunityByMatrixClient(inviteeId, communityId, client)
	if errInv != nil {
		return fmt.Errorf(
			"Failed inviting %s to %s (%s), but also failed checking invites: %s",
			inviteeId,
			communityId,
			err,
			errInv,
		)
	}

	if !isInvited {
		return fmt.Errorf(
			"Failed inviting %s to %s (%s) and determined the user to not be invited",
			inviteeId,
			communityId,
			err,
		)
	}

	// Invitation failed, but user turned out to be invited, so it's OK.
	return nil
}

func (me *ApiConnector) isUserIdInvitedToCommunityByMatrixClient(
	userId string,
	communityId string,
	client *gomatrix.Client,
) (bool, error) {
	var response matrix.ApiCommunityInvitedUsersResponse

	err := client.MakeRequest(
		"GET",
		client.BuildURL(fmt.Sprintf(
			"/groups/%s/invited_users",
			communityId,
		)),
		nil,
		&response,
	)
	if err != nil {
		return false, err
	}

	for _, user := range response.Chunk {
		if user.Id == userId {
			return true, nil
		}
	}

	return false, nil
}

func (me *ApiConnector) AcceptCommunityInvite(
	ctx *AccessTokenContext,
	userId string,
	communityId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "community.accept_invite", func() error {
		return client.MakeRequest(
			"PUT",
			client.BuildURL(fmt.Sprintf(
				"/groups/%s/self/accept_invite",
				communityId,
			)),
			map[string]interface{}{},
			nil,
		)
	})
}

func (me *ApiConnector) KickUserFromCommunity(
	ctx *AccessTokenContext,
	kickerUserId string,
	kickeeUserId string,
	communityId string,
) error {
	if kickerUserId == kickeeUserId {
		return fmt.Errorf("Kicking self (%s) does not make sense", kickerUserId)
	}

	client, err := me.createMatrixClientForUserId(ctx, kickerUserId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "community.kick", func() error {
		// This request is idempotent.
		return client.MakeRequest(
			"PUT",
			client.BuildURL(fmt.Sprintf(
				"/groups/%s/admin/users/remove/%s",
				communityId,
				kickeeUserId,
			)),
			map[string]interface{}{},
			nil,
		)
	})
}

func (me *ApiConnector) InviteUserToRoom(
	ctx *AccessTokenContext,
	inviterId string,
	inviteeId string,
	roomId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, inviterId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "room.invite", func() error {
		_, err := client.InviteUser(roomId, &gomatrix.ReqInviteUser{UserID: inviteeId})
		return err
	})
}

func (me *ApiConnector) JoinRoom(
	ctx *AccessTokenContext,
	userId string,
	roomId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "room.join", func() error {
		// This request is idempotent.
		_, err := client.JoinRoom(roomId, "", nil)
		return err
	})
}

func (me *ApiConnector) DemoteUserInRoom(
	ctx *AccessTokenContext,
	userId string,
	roomId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	var powerLevels map[string]interface{}

	err = client.StateEvent(roomId, "m.room.power_levels", "", &powerLevels)
	if err != nil {
		return err
	}

	jsonObj, err := gabs.Consume(powerLevels)
	if err != nil {
		return err
	}

	userPowerLevel, ok := jsonObj.Search("users", userId).Data().(float64)
	if !ok {
		// Most likely no power level defined for the user, which means the default applies.
		// The default is not likely to be very privileged, so let's consider this demoted enough.
		return nil
	}

	if userPowerLevel == 0 {
		// This is the lowest power level, which is ideal.
		return nil
	}

	jsonObj.Set(0, "users", userId)

	return matrix.ExecuteWithRateLimitRetries(me.logger, "user.demote", func() error {
		_, err := client.SendStateEvent(roomId, "m.room.power_levels", "", jsonObj.Data())
		return err
	})
}

func (me *ApiConnector) KickUserFromRoom(
	ctx *AccessTokenContext,
	kickerUserId string,
	kickeeUserId string,
	roomId string,
) error {
	if kickerUserId == kickeeUserId {
		return fmt.Errorf("Kicking self (%s) does not make sense", kickerUserId)
	}

	client, err := me.createMatrixClientForUserId(ctx, kickerUserId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "room.kick", func() error {
		// This request is idempotent.
		_, err := client.KickUser(roomId, &gomatrix.ReqKickUser{
			UserID: kickeeUserId,
		})
		return err
	})
}

func (me *ApiConnector) LeaveRoom(
	ctx *AccessTokenContext,
	userId string,
	roomId string,
) error {
	client, err := me.createMatrixClientForUserId(ctx, userId)
	if err != nil {
		return err
	}

	return matrix.ExecuteWithRateLimitRetries(me.logger, "room.leave", func() error {
		// This request is idempotent.
		_, err := client.LeaveRoom(roomId)
		return err
	})
}

// createMatrixClientForUserId gets an access token (reuses or obtains a new one) for the user
// and creates an API client with it
func (me *ApiConnector) createMatrixClientForUserId(
	ctx *AccessTokenContext,
	userId string,
) (*gomatrix.Client, error) {
	accessToken, err := ctx.GetAccessTokenForUserId(userId)
	if err != nil {
		return nil, err
	}

	return me.createMatrixClientForUserIdAndToken(userId, accessToken)
}

func (me *ApiConnector) createMatrixClientForUserIdAndToken(
	userId string,
	accessToken string,
) (*gomatrix.Client, error) {
	client, err := gomatrix.NewClient(me.homeserverApiEndpoint, userId, accessToken)
	if err != nil {
		err = fmt.Errorf("Failed creating client for %s: %s", userId, err)
	}

	client.Client = me.httpClient

	return client, err
}

// VerifyAccessToken verifies that an access token works and belongs
// to the user it's expected to belong to
func (me *ApiConnector) VerifyAccessToken(userId string, accessToken string) error {
	client, err := gomatrix.NewClient(me.homeserverApiEndpoint, userId, accessToken)
	if err != nil {
		return err
	}

	var resp matrix.ApiWhoAmIResponse

	err = client.MakeRequest("GET", client.BuildURL("/account/whoami"), nil, &resp)
	if err != nil {
		return err
	}

	if resp.UserId != userId {
		return fmt.Errorf("Failed who-am-I user id verification check")
	}

	return nil
}
