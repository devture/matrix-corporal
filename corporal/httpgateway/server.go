package httpgateway

import (
	"context"
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httpgateway/policycheck"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	logger              *logrus.Logger
	configuration       configuration.HttpGateway
	reverseProxy        *httputil.ReverseProxy
	userMappingResolver *matrix.UserMappingResolver
	policyStore         *policy.Store
	policyChecker       *policy.Checker
	hookRunner          *HookRunner
	catchAllHandler     *CatchAllHandler
	loginInterceptor    Interceptor
	writeTimeout        time.Duration

	server *http.Server
}

func NewServer(
	logger *logrus.Logger,
	configuration configuration.HttpGateway,
	reverseProxy *httputil.ReverseProxy,
	userMappingResolver *matrix.UserMappingResolver,
	policyStore *policy.Store,
	policyChecker *policy.Checker,
	hookRunner *HookRunner,
	catchAllHandler *CatchAllHandler,
	loginInterceptor Interceptor,
	writeTimeout time.Duration,
) *Server {
	return &Server{
		logger:              logger,
		configuration:       configuration,
		reverseProxy:        reverseProxy,
		userMappingResolver: userMappingResolver,
		policyStore:         policyStore,
		policyChecker:       policyChecker,
		hookRunner:          hookRunner,
		catchAllHandler:     catchAllHandler,
		loginInterceptor:    loginInterceptor,
		writeTimeout:        writeTimeout,

		server: nil,
	}
}

func (me *Server) Start() error {
	me.server = &http.Server{
		Handler:      me.createRouter(),
		Addr:         me.configuration.ListenAddress,
		WriteTimeout: me.writeTimeout,
		ReadTimeout:  10 * time.Second,
	}

	me.logger.Infof("Starting HTTP Gateway Server on %s", me.server.Addr)

	go func() {
		err := me.server.ListenAndServe()
		if err != http.ErrServerClosed {
			me.logger.Panicf("HTTP Gateway Server error: %s", err)
		}
	}()

	return nil
}

func (me *Server) Stop() error {
	if me.server == nil {
		return nil
	}

	me.logger.Infoln("Stopping HTTP Gateway Server")
	me.server.Shutdown(context.Background())

	return nil
}

func (me *Server) createRouter() http.Handler {
	r := mux.NewRouter()

	r.Use(denyUnsupportedApiVersionsMiddleware)

	// To make it easy to detect if Matrix Corporal is properly fronting the Matrix Client-Server API,
	// we add this custom non-standard route.
	r.HandleFunc(
		"/_matrix/client/corporal",
		func(w http.ResponseWriter, r *http.Request) {
			logger := me.logger.WithField("method", r.Method)
			logger = logger.WithField("uri", r.RequestURI)
			logger.Debugf("HTTP gateway: serving Matrix Corporal info page")

			w.Write([]byte("Matrix Client-Server API protected by Matrix Corporal"))
		},
	).Methods("GET")

	r.HandleFunc(
		"/_matrix/client/r0/groups/{communityId}/self/leave",
		me.createPolicyCheckingHandler("community.self.leave", policycheck.CheckCommunitySelfLeave),
	).Methods("PUT")

	r.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/leave",
		me.createPolicyCheckingHandler("room.leave", policycheck.CheckRoomLeave),
	).Methods("POST")

	// Another way to leave a room is to PUT a "membership=leave" into your m.room.member state.
	r.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/state/m.room.member/{memberId}",
		me.createPolicyCheckingHandler("room.member.state.set", policycheck.CheckRoomMembershipStateChange),
	).Methods("PUT")

	// Another way to leave a room is kick yourself out of it. It doesn't require any special permissions.
	r.HandleFunc(
		"/_matrix/client/r0/rooms/{roomId}/kick",
		me.createPolicyCheckingHandler("room.kick", policycheck.CheckRoomKick),
	).Methods("POST")

	r.HandleFunc(
		"/_matrix/client/r0/createRoom",
		me.createPolicyCheckingHandler("room.create", policycheck.CheckRoomCreate),
	).Methods("POST")

	r.HandleFunc(
		"/_matrix/client/r0/profile/{targetUserId}/displayname",
		me.createPolicyCheckingHandler("user.set_display_name", policycheck.CheckProfileSetDisplayName),
	).Methods("PUT")

	r.HandleFunc(
		"/_matrix/client/r0/profile/{targetUserId}/avatar_url",
		me.createPolicyCheckingHandler("user.set_avatar", policycheck.CheckProfileSetAvatarUrl),
	).Methods("PUT")

	r.HandleFunc(
		"/_matrix/client/r0/account/deactivate",
		me.createPolicyCheckingHandler("user.deactivate", policycheck.CheckUserDeactivate),
	).Methods("POST")

	r.HandleFunc(
		"/_matrix/client/r0/account/password",
		me.createPolicyCheckingHandler("user.password", policycheck.CheckUserSetPassword),
	).Methods("POST")

	r.Handle(
		"/_matrix/client/r0/login",
		me.createInterceptorHandler("login", me.loginInterceptor),
	).Methods("POST")

	r.PathPrefix("/").Handler(me.catchAllHandler)

	return r
}

func (me *Server) createPolicyCheckingHandler(name string, policyCheckingCallback policycheck.PolicyCheckFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)
		logger = logger.WithField("handler", name)

		httpResponseModifierFuncs := make([]hook.HttpResponseModifierFunc, 0)

		hookResult := me.hookRunner.RunFirstMatchingType(hook.EventTypeBeforeAnyRequest, w, r, logger)
		if hookResult.SkipProceedingFurther {
			logger.Debugf("HTTP gateway (policy-checked): %s hook said we should not proceed further", hook.EventTypeBeforeAnyRequest)
			return
		}
		if hookResult.ReverseProxyResponseModifier != nil {
			httpResponseModifierFuncs = append(httpResponseModifierFuncs, *hookResult.ReverseProxyResponseModifier)
		}

		accessToken := httphelp.GetAccessTokenFromRequest(r)
		if accessToken == "" {
			logger.Debugf("HTTP gateway (policy-checked): rejecting (missing access token)")

			httphelp.RespondWithMatrixError(
				w,
				http.StatusUnauthorized,
				matrix.ErrorMissingToken,
				"Missing access token",
			)
			return
		}

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

		// These will be read in handlers and in hooks (like `hook.EventTypeBeforeAuthenticatedPolicyCheckedRequest`).
		r = r.WithContext(context.WithValue(r.Context(), "accessToken", accessToken))
		r = r.WithContext(context.WithValue(r.Context(), "userId", userId))

		hookResult = me.hookRunner.RunFirstMatchingType(hook.EventTypeBeforeAuthenticatedPolicyCheckedRequest, w, r, logger)
		if hookResult.SkipProceedingFurther {
			logger.Debugf("HTTP gateway (policy-checked): %s hook said we should not proceed further", hook.EventTypeBeforeAuthenticatedPolicyCheckedRequest)
			return
		}
		if hookResult.ReverseProxyResponseModifier != nil {
			httpResponseModifierFuncs = append(httpResponseModifierFuncs, *hookResult.ReverseProxyResponseModifier)
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

		reverseProxyToUse := me.reverseProxy

		if len(httpResponseModifierFuncs) == 0 {
			logger.Debugf("HTTP gateway (policy-checked): proxying")
		} else {
			logger.Debugf("HTTP gateway (policy-checked): proxying (with response modification)")

			var chainedResponseModifier hook.HttpResponseModifierFunc = func(response *http.Response) error {
				for _, httpResponseModifierFunc := range httpResponseModifierFuncs {
					err := httpResponseModifierFunc(response)
					if err != nil {
						return err
					}
				}
				return nil
			}

			reverseProxyCopy := *reverseProxyToUse
			reverseProxyCopy.ModifyResponse = chainedResponseModifier

			reverseProxyToUse = &reverseProxyCopy
		}

		reverseProxyToUse.ServeHTTP(w, r)
	}
}

func (me *Server) createInterceptorHandler(name string, interceptor Interceptor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interceptorResult := interceptor.Intercept(r)

		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)
		logger = logger.WithField("handler", name)
		logger = logger.WithFields(interceptorResult.LoggingContextFields)

		if interceptorResult.Result == InterceptorResultProxy {
			logger.Infof("HTTP gateway (intercepted): proxying")

			me.reverseProxy.ServeHTTP(w, r)

			return
		}

		if interceptorResult.Result == InterceptorResultDeny {
			logger.Infof(
				"HTTP gateway (intercepted): denying (%s: %s)",
				interceptorResult.ErrorCode,
				interceptorResult.ErrorMessage,
			)

			httphelp.RespondWithMatrixError(
				w,
				http.StatusForbidden,
				interceptorResult.ErrorCode,
				interceptorResult.ErrorMessage,
			)

			return
		}

		logger.Fatalf("HTTP gateway (intercepted): unexpected interceptor result: %#v", interceptorResult)
	}
}
