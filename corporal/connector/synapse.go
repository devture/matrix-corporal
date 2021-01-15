package connector

import (
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/util"
	"fmt"
	"sync"
	"time"

	"crypto/hmac"
	"crypto/sha1"

	"github.com/matrix-org/gomatrix"
)

const (
	deviceIdCorporal = "matrix-corporal"
)

// SynapseConnector is a MatrixConnector implementation for controlling a Synapse server.
// It is based on the base ApiConnector for doing whatever's possible,
// but also contains Synapse-specific API calls here.
type SynapseConnector struct {
	*ApiConnector

	registrationSharedSecret string
	corporalUserID           string

	corporalUserAccessTokenContext *AccessTokenContext

	corporalUserIDLock *sync.Mutex
}

func NewSynapseConnector(
	apiConnector *ApiConnector,
	registrationSharedSecret string,
	corporalUserID string,
) *SynapseConnector {
	me := &SynapseConnector{
		ApiConnector: apiConnector,

		registrationSharedSecret: registrationSharedSecret,
		corporalUserID:           corporalUserID,

		corporalUserIDLock: &sync.Mutex{},
	}

	// This is a special access token context that we only use for the matrix-corporal user.
	// We force it to use the ApiConnector (and not this SynapseConnector),
	// because we wish to obtain a token for that user directly and not via the custom admin APIs in `ObtainNewAccessTokenForUserId()` below.
	me.corporalUserAccessTokenContext = NewAccessTokenContext(
		me.ApiConnector,
		deviceIdCorporal,
		// Using a validity of 0, because we never want this token to expire.
		// We release it manually from `Release()`.
		0,
	)

	return me
}

// ObtainNewAccessTokenForUserId is a reimplementation of ApiConnector.ObtainNewAccessTokenForUserId.
//
// ApiConnector.ObtainNewAccessTokenForUserId uses the regular `/_matrix/client/r0/login` endpoint
// and relies on shared-secret-auth to impersonate a user.
//
// This implementation here relies on an admin's access token and on the `POST /_synapse/admin/v1/users/<user_id>/login` API
// (see https://github.com/matrix-org/synapse/pull/8617), to obtain a non-device-creating token for any user.
//
// Not creating devices leads to better performance and UX (no need to notify others via federation; the user's device list does not get poluted).
// This is Synapse-specific though.
func (me *SynapseConnector) ObtainNewAccessTokenForUserId(userId, deviceId string, validUntil *time.Time) (string, error) {
	if userId == me.corporalUserID {
		// Someone explicitly requested a token for the matrix-corporal user.
		// If we try to proceed below (using the Admin user login API to log in as matrix-corporal),
		// we'll hit an error: "Cannot use admin API to login as self".
		//
		// Requests to log in as the matrix-corporal user should be handled separately.
		//
		// We may wish to use `me.getAccessTokenForCorporalUser()` and just return that, but that's
		// also not good enough, because we use this token for our own internal purposes and we want it
		// to remain valid until we dispose of it ourselves.
		// Giving it out to consumers means that they may destroy it (and our reconciliation code will certainly do that during cleanup!).
		// Having this token end up destroyed will break all future invocations of this function.
		//
		// So.. we don't reuse an existing token, but always obtain a fresh one.
		return me.ApiConnector.ObtainNewAccessTokenForUserId(userId, deviceId, validUntil)
	}

	corporalUserAccessToken, err := me.getAccessTokenForCorporalUser()
	if err != nil {
		return "", fmt.Errorf(
			"Could not obtain access token for `%s`, necessary for obtaining a token for `%s`: %s",
			me.corporalUserID,
			userId,
			err,
		)
	}

	client, err := me.createMatrixClientForUserIdAndToken(me.corporalUserID, corporalUserAccessToken)
	if err != nil {
		return "", err
	}

	requestPayload := map[string]interface{}{}
	if validUntil != nil {
		requestPayload["valid_until_ms"] = validUntil.Unix() * 1000
	}

	var response matrix.ApiAdminResponseUserLogin
	err = client.MakeRequest(
		"POST",
		buildPrefixlessURL(client, fmt.Sprintf("/_synapse/admin/v1/users/%s/login", userId), map[string]string{}),
		requestPayload,
		&response,
	)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (me *SynapseConnector) DetermineCurrentState(
	ctx *AccessTokenContext,
	managedUserIds []string,
	adminUserId string,
) (*CurrentState, error) {
	client, err := me.createMatrixClientForUserId(ctx, adminUserId)
	if err != nil {
		return nil, err
	}

	// The `/_synapse/admin/v2/users` API lets us use the `user_id` parameter
	// to query for individual users.
	//
	// Instead of looping through all users on the server (potentially millions?),
	// we could loop over the managedUserIds, and fetch the state for them,
	// instead of fetching all users (like we do below).
	//
	// On a server with millions of unmanaged users and a subset of managed users,
	// it's more beneficial to do it selectively.
	//
	// On a server where pretty much all users are managed users and there are lots of them,
	// it's better to avoid doing an individual query for each managed

	url := buildPrefixlessURL(client, "/_synapse/admin/v2/users", map[string]string{
		// We don't support pagination yet
		"limit":       "100000000000",
		"guests":      "false",
		"deactivated": "true",
	})

	var response matrix.ApiAdminResponseUsers
	err = client.MakeRequest("GET", url, nil, &response)
	if err != nil {
		return nil, err
	}

	var currentUserIds []string
	for _, user := range response.Users {
		currentUserIds = append(currentUserIds, user.Id)
	}

	var usersState []CurrentUserState

	for _, userId := range managedUserIds {
		if !util.IsStringInArray(userId, currentUserIds) {
			// Avoid trying to fetch the state for a user that doesn't exist.
			// We'll get authentication errors.
			// And it's not like there could be any state anyway, so.. skip it.
			continue
		}

		userState, err := me.getUserStateByUserId(ctx, userId)
		if err != nil {
			return nil, err
		}
		usersState = append(usersState, *userState)
	}

	connectorState := &CurrentState{
		Users: usersState,
	}

	return connectorState, nil
}

func (me *SynapseConnector) EnsureUserAccountExists(userId, password string) error {
	userIdLocalPart, err := gomatrix.ExtractUserLocalpart(userId)
	if err != nil {
		return err
	}

	client, _ := gomatrix.NewClient(me.homeserverApiEndpoint, "", "")

	var nonceResponse matrix.ApiUserAccountRegisterNonceResponse
	err = matrix.ExecuteWithRateLimitRetries(me.logger, "user.register.nonce", func() error {
		return client.MakeRequest(
			"GET",
			buildPrefixlessURL(client, "/_synapse/admin/v1/register", map[string]string{}),
			nil,
			&nonceResponse,
		)
	})
	if err != nil {
		return err
	}

	// Generating the HMAC the same way that the `register_new_matrix_user` script from Matrix Synapse does it.
	mac := hmac.New(sha1.New, []byte(me.registrationSharedSecret))
	mac.Write([]byte(nonceResponse.Nonce))
	mac.Write([]byte("\x00"))
	mac.Write([]byte(userIdLocalPart))
	mac.Write([]byte("\x00"))
	mac.Write([]byte(password))
	mac.Write([]byte("\x00"))
	mac.Write([]byte("notadmin"))

	payload := matrix.ApiUserAccountRegisterRequestPayload{
		Nonce:    nonceResponse.Nonce,
		Username: userIdLocalPart,
		Password: password,
		Mac:      fmt.Sprintf("%x", mac.Sum(nil)),
		Type:     matrix.RegistrationTypeSharedSecret,
		Admin:    false,
	}

	var registerResponse matrix.ApiUserAccountRegisterResponse

	err = matrix.ExecuteWithRateLimitRetries(me.logger, "user.register.actual", func() error {
		// The canonical admin/register API is available at `/_synapse/admin/v1/register`.
		// What we hit below is an alias, which might stop working some time in the future.
		// See above for why we can't easily use it.
		return client.MakeRequest(
			"POST",
			buildPrefixlessURL(client, "/_synapse/admin/v1/register", map[string]string{}),
			payload,
			&registerResponse,
		)
	})

	if err != nil {
		// Swallow "user already exists" errors.
		// We don't care who created it and when. We only care that it exists.
		if matrix.IsErrorWithCode(err, matrix.ErrorUserInUse) {
			return nil
		}

		return err
	}

	// The register API creates an access token automatically.
	// We don't need it and we'd rather be nice and get rid of it, to keep things clean.
	clientForUser, _ := gomatrix.NewClient(me.homeserverApiEndpoint, userIdLocalPart, registerResponse.AccessToken)
	clientForUser.Logout()

	return nil
}

func (me *SynapseConnector) Release() {
	me.corporalUserAccessTokenContext.Release()
}

func (me *SynapseConnector) getAccessTokenForCorporalUser() (string, error) {
	me.corporalUserIDLock.Lock()
	defer me.corporalUserIDLock.Unlock()

	return me.corporalUserAccessTokenContext.GetAccessTokenForUserId(me.corporalUserID)
}
