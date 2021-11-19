package handler

import (
	"context"
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httpgateway/hookrunner"
	"devture-matrix-corporal/corporal/httpgateway/policycheck"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type policyCheckedRoutesHandler struct {
	reverseProxy        *httputil.ReverseProxy
	policyStore         *policy.Store
	policyChecker       *policy.Checker
	hookRunner          *hookrunner.HookRunner
	userMappingResolver *matrix.UserMappingResolver
	logger              *logrus.Logger
}

func NewPolicyCheckedRoutesHandler(
	reverseProxy *httputil.ReverseProxy,
	policyStore *policy.Store,
	policyChecker *policy.Checker,
	hookRunner *hookrunner.HookRunner,
	userMappingResolver *matrix.UserMappingResolver,
	logger *logrus.Logger,
) *policyCheckedRoutesHandler {
	return &policyCheckedRoutesHandler{
		reverseProxy:        reverseProxy,
		policyStore:         policyStore,
		policyChecker:       policyChecker,
		hookRunner:          hookRunner,
		userMappingResolver: userMappingResolver,
		logger:              logger,
	}
}

func (me *policyCheckedRoutesHandler) RegisterRoutesWithRouter(router *mux.Router) {
	// All routes below define an optional trailing slash.
	//
	// It's important to us that policy-checked routes are matched (with and without a slash),
	// so we can guarantee policy checking happens and that we potentially reject requests that need to be rejected.
	// Without this, a request for `/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/state/m.room.encryption/` (note the trailing slash)
	// does not match our `/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/state/m.room.encryption` policy-checked handler,
	// slips through and gets happily served by the homserver.
	//
	// Alternative solutions are:
	// 1. Using the mux Router's `StripSlash(true)` setting (but it does weird 301 redirects for POST/PUT requests, so it's not effective)
	// 2. Applying a router middleware that modifies the request (stripping trailing slashes) before matching happens.
	// See: https://natedenlinger.com/dealing-with-trailing-slashes-on-requesturi-in-go-with-mux/
	// The 2nd solution doesn't work as well, because some APIs (`GET /_matrix/client/{apiVersion:(?:r0|v\d+)}/pushrules/`) require a trailing slash.
	// Removing the trailing slash on our side and forwarding the request to the homeserver results in `{"errcode":"M_UNRECOGNIZED","error":"Unrecognized request"}`.
	//
	// Instead of trying to whitelist routes that require a slash and potentially missing something,
	// we instead make matching for our policy-checked routes tolerate a trailing slash.
	//
	// Most of the APIs below will not even be served by the homeserver, as a trailing slash is not tolerated at the homeserver level.
	// Still, it's safer if we policy-check them all and not have to worry future homeserver versions handling things differently.

	// Requests for an `apiVersion` that we don't support (and don't match below) are rejected via a `denyUnsupportedApiVersionsMiddleware` middleware.

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/groups/{communityId}/self/leave{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("community.self.leave", policycheck.CheckCommunitySelfLeave, false),
	).Methods("PUT")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/leave{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.leave", policycheck.CheckRoomLeave, false),
	).Methods("POST")

	// Another way to leave a room is kick yourself out of it. It doesn't require any special permissions.
	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/kick{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.kick", policycheck.CheckRoomKick, false),
	).Methods("POST")

	// Another way to leave a room is to PUT a "membership=leave" into your m.room.member state.
	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/state/m.room.member/{memberId}{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.member.state.set", policycheck.CheckRoomMembershipStateChange, false),
	).Methods("PUT")

	// Another way to make a room encrypted is by enabling encryption subsequently.
	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/state/m.room.encryption{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.subsequenly_enabling_encryption", policycheck.CheckRoomEncryptionStateChange, false),
	).Methods("PUT")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/createRoom{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.create", policycheck.CheckRoomCreate, false),
	).Methods("POST")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/rooms/{roomId}/send/{eventType}/{txnId}{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("room.send_event", policycheck.CheckRoomSendEvent, false),
	).Methods("PUT")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/profile/{targetUserId}/displayname{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("user.set_display_name", policycheck.CheckProfileSetDisplayName, false),
	).Methods("PUT")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/profile/{targetUserId}/avatar_url{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("user.set_avatar", policycheck.CheckProfileSetAvatarUrl, false),
	).Methods("PUT")

	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/account/deactivate{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("user.deactivate", policycheck.CheckUserDeactivate, false),
	).Methods("POST")

	// This Client-Server API is used for 2 things:
	// - setting new passwords for authenticated users (requests having an access token)
	// - a "forgotten password" flow for unauthenticated users (they authenticate by verifying some 3pid)
	//
	// We don't want to break the 2nd (access-token-less) flow in some cases (depending on the policy).
	router.HandleFunc(
		`/_matrix/client/{apiVersion:(?:r0|v\d+)}/account/password{optionalTrailingSlash:[/]?}`,
		me.createPolicyCheckingHandler("user.password", policycheck.CheckUserSetPassword, true),
	).Methods("POST")
}

func (me *policyCheckedRoutesHandler) createPolicyCheckingHandler(
	name string,
	policyCheckingCallback policycheck.PolicyCheckFunc,
	allowUnauthenticatedAccess bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)
		logger = logger.WithField("handler", name)

		httpResponseModifierFuncs := make([]hook.HttpResponseModifierFunc, 0)

		if !runHooks(me.hookRunner, hook.EventTypeBeforeAnyRequest, w, r, logger, &httpResponseModifierFuncs) {
			return
		}

		// Depending on the route, we may or may not allow requests having no access token to go through.
		accessToken := httphelp.GetAccessTokenFromRequest(r)
		if accessToken == "" {
			if allowUnauthenticatedAccess {
				logger.Debugf("HTTP gateway (policy-checked): missing token, but allowing request to go through")
			} else {
				logger.Debugf("HTTP gateway (policy-checked): rejecting (missing access token)")

				httphelp.RespondWithMatrixError(
					w,
					http.StatusUnauthorized,
					matrix.ErrorMissingToken,
					"Missing access token",
				)
				return
			}
		}

		isAuthenticated := false

		// However, if there is an access token, we'd require it be a valid one (successfully mapping to a user).
		if accessToken != "" {
			userId, err := me.userMappingResolver.ResolveByAccessToken(accessToken)
			if err != nil {
				logger.Debugf("HTTP gateway (policy-checked): rejecting (failed to map access token)")

				httphelp.RespondWithMatrixError(
					w,
					http.StatusForbidden,
					matrix.ErrorUnknownToken,
					"Failed mapping access token to user id",
				)
				return
			}
			logger = logger.WithField("userId", userId)

			// These will be read in handlers and in hooks (like `hook.EventTypeBeforeAuthenticatedRequest`).
			r = r.WithContext(context.WithValue(r.Context(), "accessToken", accessToken))
			r = r.WithContext(context.WithValue(r.Context(), "userId", userId))

			isAuthenticated = true
		}

		if isAuthenticated {
			if !runHooks(me.hookRunner, hook.EventTypeBeforeAuthenticatedRequest, w, r, logger, &httpResponseModifierFuncs) {
				return
			}

			if !runHooks(me.hookRunner, hook.EventTypeBeforeAuthenticatedPolicyCheckedRequest, w, r, logger, &httpResponseModifierFuncs) {
				return
			}
		}

		policy := me.policyStore.Get()
		if policy == nil {
			logger.Infof("HTTP gateway (policy-checked): denying (missing policy)")

			httphelp.RespondWithMatrixError(
				w,
				http.StatusForbidden,
				matrix.ErrorForbidden,
				"Policy does not exist (yet), so access cannot be allowed",
			)
			return
		}

		policyResponse := policyCheckingCallback(r, r.Context(), *policy, *me.policyChecker)

		if !policyResponse.Allow {
			logger.Infof(
				"HTTP gateway (policy-checked): denying (%s: %s)",
				policyResponse.ErrorCode,
				policyResponse.ErrorMessage,
			)

			httphelp.RespondWithMatrixError(
				w,
				http.StatusForbidden,
				policyResponse.ErrorCode,
				policyResponse.ErrorMessage,
			)
			return
		}

		if !runHooks(me.hookRunner, hook.EventTypeAfterAnyRequest, w, r, logger, &httpResponseModifierFuncs) {
			return
		}

		if isAuthenticated {
			if !runHooks(me.hookRunner, hook.EventTypeAfterAuthenticatedRequest, w, r, logger, &httpResponseModifierFuncs) {
				return
			}

			if !runHooks(me.hookRunner, hook.EventTypeAfterAuthenticatedPolicyCheckedRequest, w, r, logger, &httpResponseModifierFuncs) {
				return
			}
		}

		reverseProxyToUse := me.reverseProxy

		if len(httpResponseModifierFuncs) == 0 {
			logger.Debugf("HTTP gateway (policy-checked): proxying")
		} else {
			logger.Debugf("HTTP gateway (policy-checked): proxying (with response modification)")

			reverseProxyCopy := *reverseProxyToUse
			reverseProxyCopy.ModifyResponse = hook.CreateChainedHttpResponseModifierFunc(httpResponseModifierFuncs)
			reverseProxyToUse = &reverseProxyCopy
		}

		reverseProxyToUse.ServeHTTP(w, r)
	}
}

// Ensure interface is implemented
var _ httphelp.HandlerRegistrator = &policyCheckedRoutesHandler{}
