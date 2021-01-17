package userauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// RestAuthenticator is a user authenticator which verifies credentials with a remote server via a REST HTTP call.
//
// The remote server's HTTP API endpoint which verifies credentials
// would be specified in the `authCredential` argument passed to Authenticate().
//
// The REST HTTP endpoint needs to handle requests and provide responses,
// with a format identical to the one used by [matrix-synapse-rest-auth](https://github.com/kamax-io/matrix-synapse-rest-auth).
//
// Note: the request/response format didn't have to be like that and no part of this project
// actually requires that `matrix-synapse-rest-auth` is installed and used.
// We just reuse the same data format for compatibility reasons and so that people who had
// previously implemented `matrix-synapse-rest-auth` could easily bridge with us.
type RestAuthenticator struct {
}

func NewRestAuthenticator() *RestAuthenticator {
	return &RestAuthenticator{}
}

func (me *RestAuthenticator) Type() string {
	return "rest"
}

func (me *RestAuthenticator) Authenticate(userId, givenPassword, authCredential string) (bool, error) {
	// authCredential is actually expected to be a URL where the given user is to be authenticated.
	// This URL gets passed down to us from the user's policy.
	restAuthApiUrl := authCredential

	payload := NewRestAuthRequest(userId, givenPassword)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	response, err := http.Post(restAuthApiUrl, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return false, err
	}

	if response.StatusCode != 200 {
		return false, fmt.Errorf("Non-OK HTTP response for %s: %d", restAuthApiUrl, response.StatusCode)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	var authResult RestAuthResponse
	err = json.Unmarshal(responseBytes, &authResult)
	if err != nil {
		return false, fmt.Errorf("Failed to decode JSON (%s) for %s: %s", err, restAuthApiUrl, responseBytes)
	}

	return authResult.Auth.Success, nil
}

type restAuthRequest struct {
	User restAuthRequestUser `json:"user"`
}

func NewRestAuthRequest(userId, password string) restAuthRequest {
	return restAuthRequest{
		User: restAuthRequestUser{
			Id:       userId,
			Password: password,
		},
	}
}

type restAuthRequestUser struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

type RestAuthResponse struct {
	Auth RestAuthResponseAuth `json:"auth"`
}

func NewUnsuccessfulRestAuthResponse() RestAuthResponse {
	return RestAuthResponse{
		Auth: RestAuthResponseAuth{
			Success: false,
		},
	}
}

type RestAuthResponseAuth struct {
	Success bool `json:"success"`

	MatrixID string `json:"mxid,omitempty"`

	Profile *RestAuthResponseAuthProfile `json:"profile,omitempty"`

	// As per the documentation for `matrix-synapse-rest-auth`,
	// additional fields might appear here.
	// We don't use or care about them, so we ignore everything else.
}

type RestAuthResponseAuthProfile struct {
	DisplayName string `json:"display_name"`
}
