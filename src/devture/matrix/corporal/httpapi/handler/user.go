package handler

import (
	"devture/matrix/corporal/connector"
	"devture/matrix/corporal/httphelp"
	"devture/matrix/corporal/matrix"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// apiAccessTokenObtainRequestPayload is a request payload for: POST /_matrix/corporal/user/{userId}/access-token/obtain
type apiAccessTokenObtainRequestPayload struct {
	DeviceId string `json:"deviceId"`
}

// apiAccessTokenObtainRequestPayload is a response for: POST /_matrix/corporal/user/{userId}/access-token/obtain
type apiAccessTokenObtainResponse struct {
	ApiResponse
	AccessToken string `json:"accessToken"`
}

// apiAccessTokenReleaseRequestPayload is a request payload for: DELETE /_matrix/corporal/user/{userId}/access-token
type apiAccessTokenReleaseRequestPayload struct {
	AccessToken string `json:"accessToken"`
}

type UserApiHandlerRegistrator struct {
	homeserverDomainName string
	connector            connector.MatrixConnector
}

func NewUserApiHandlerRegistrator(
	homeserverDomainName string,
	connector connector.MatrixConnector,
) *UserApiHandlerRegistrator {
	return &UserApiHandlerRegistrator{
		homeserverDomainName: homeserverDomainName,
		connector:            connector,
	}
}

func (me *UserApiHandlerRegistrator) RegisterRoutesWithRouter(router *mux.Router) {
	router.HandleFunc("/_matrix/corporal/user/{userId}/access-token", me.actionAccessTokenRelease).Methods("DELETE")
	router.HandleFunc("/_matrix/corporal/user/{userId}/access-token/new", me.actionAccessTokenObtain).Methods("POST")
}

func (me *UserApiHandlerRegistrator) actionAccessTokenObtain(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]

	if !matrix.IsFullUserIdOfDomain(userId, me.homeserverDomainName) {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok: false,
			Error: fmt.Sprintf(
				"Bad user id (%s) - not part of the homeserver domain (%s)",
				userId,
				me.homeserverDomainName,
			),
		})
		return
	}

	var payload apiAccessTokenObtainRequestPayload

	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok:    false,
			Error: "Bad body payload",
		})
		return
	}

	if payload.DeviceId == "" {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok:    false,
			Error: "Bad body payload - empty or missing device id",
		})
		return
	}

	accessToken, err := me.connector.ObtainNewAccessTokenForUserId(userId, payload.DeviceId)
	if err != nil {
		Respond(w, http.StatusServiceUnavailable, ApiResponse{
			Ok: false,
			Error: fmt.Sprintf(
				"Could not obtain access token: %s",
				err,
			),
		})
		return
	}

	Respond(w, http.StatusOK, apiAccessTokenObtainResponse{
		ApiResponse: ApiResponse{
			Ok: true,
		},
		AccessToken: accessToken,
	})
}

func (me *UserApiHandlerRegistrator) actionAccessTokenRelease(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]

	if !matrix.IsFullUserIdOfDomain(userId, me.homeserverDomainName) {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok: false,
			Error: fmt.Sprintf(
				"Bad user id (%s) - not part of the homeserver domain (%s)",
				userId,
				me.homeserverDomainName,
			),
		})
		return
	}

	var payload apiAccessTokenReleaseRequestPayload

	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok:    false,
			Error: "Bad body payload",
		})
		return
	}

	if payload.AccessToken == "" {
		Respond(w, http.StatusBadRequest, ApiResponse{
			Ok:    false,
			Error: "Bad body payload - empty or missing access token",
		})
		return
	}

	// This is idempotent.
	err = me.connector.DestroyAccessToken(userId, payload.AccessToken)
	if err != nil {
		Respond(w, http.StatusServiceUnavailable, ApiResponse{
			Ok:    false,
			Error: "Could not destroy access token",
		})
		return
	}

	Respond(w, http.StatusOK, ApiResponse{
		Ok: true,
	})
}
