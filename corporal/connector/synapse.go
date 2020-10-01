package connector

import (
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/util"
	"fmt"
	"strings"

	"crypto/hmac"
	"crypto/sha1"

	"github.com/matrix-org/gomatrix"
)

// SynapseConnector is a MatrixConnector implementation for controlling a Synapse server.
// It is based on the base ApiConnector for doing whatever's possible,
// but also contains Synapse-specific API calls here.
type SynapseConnector struct {
	*ApiConnector
	registrationSharedSecret string
}

func NewSynapseConnector(
	apiConnector *ApiConnector,
	registrationSharedSecret string,
) *SynapseConnector {
	return &SynapseConnector{
		ApiConnector:             apiConnector,
		registrationSharedSecret: registrationSharedSecret,
	}
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

	url := client.BuildURLWithQuery([]string{"/_synapse/admin/v2/users"}, map[string]string{
		// We don't support pagination yet
		"limit":       "100000000000",
		"guests":      "false",
		"deactivated": "true",
	})
	// The URL-building function above forces us under the `/_matrix/client/r0/` prefix.
	// We'd like to work at the top-level though, hence this hack.
	url = strings.Replace(url, "/_matrix/client/r0/", "/", 1)

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
		// The canonical admin/register API is available at `/_synapse/admin/v1/register`.
		// What we hit below is an alias, which might stop working some time in the future.
		//
		// We can't hit the canonical URL easily, because gomatrix insists on pre-pending
		// `/_matrix/client/r0` to URLs built via `BuildURL()`.
		return client.MakeRequest(
			"GET",
			client.BuildURL("admin/register"),
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
			client.BuildURL("admin/register"),
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
