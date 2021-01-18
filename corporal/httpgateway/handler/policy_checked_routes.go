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
	router.HandleFunc(
		"/_matrix/client/r0/groups/{communityId}/self/leave",
		me.createPolicyCheckingHandler("community.self.leave", policycheck.CheckCommunitySelfLeave, false),
	).Methods("PUT")

	router.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/leave",
		me.createPolicyCheckingHandler("room.leave", policycheck.CheckRoomLeave, false),
	).Methods("POST")

	// Another way to leave a room is kick yourself out of it. It doesn't require any special permissions.
	router.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/kick",
		me.createPolicyCheckingHandler("room.kick", policycheck.CheckRoomKick, false),
	).Methods("POST")

	// Another way to leave a room is to PUT a "membership=leave" into your m.room.member state.
	router.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/state/m.room.member/{memberId}",
		me.createPolicyCheckingHandler("room.member.state.set", policycheck.CheckRoomMembershipStateChange, false),
	).Methods("PUT")

	// Another way to make a room encrypted is by enabling encryption subsequently.
	router.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/state/m.room.encryption",
		me.createPolicyCheckingHandler("room.subsequenly_enabling_encryption", policycheck.CheckRoomEncryptionStateChange, false),
	).Methods("PUT")

	router.HandleFunc(
		"/_matrix/client/r0/createRoom",
		me.createPolicyCheckingHandler("room.create", policycheck.CheckRoomCreate, false),
	).Methods("POST")

	router.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/send/{eventType}/{txnId}",
		me.createPolicyCheckingHandler("room.send_event", policycheck.CheckRoomSendEvent, false),
	).Methods("PUT")

	router.HandleFunc(
		"/_matrix/client/r0/profile/{targetUserId}/displayname",
		me.createPolicyCheckingHandler("user.set_display_name", policycheck.CheckProfileSetDisplayName, false),
	).Methods("PUT")

	router.HandleFunc(
		"/_matrix/client/r0/profile/{targetUserId}/avatar_url",
		me.createPolicyCheckingHandler("user.set_avatar", policycheck.CheckProfileSetAvatarUrl, false),
	).Methods("PUT")

	router.HandleFunc(
		"/_matrix/client/r0/account/deactivate",
		me.createPolicyCheckingHandler("user.deactivate", policycheck.CheckUserDeactivate, false),
	).Methods("POST")

	// This Client-Server API is used for 2 things:
	// - setting new passwords for authenticated users (requests having an access token)
	// - a "forgotten password" flow for unauthenticated users (they authenticate by verifying some 3pid)
	//
	// We don't want to break the 2nd (access-token-less) flow in some cases (depending on the policy).
	router.HandleFunc(
		"/_matrix/client/r0/account/password",
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
