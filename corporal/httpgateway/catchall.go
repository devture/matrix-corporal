package httpgateway

import (
	"context"
	"devture-matrix-corporal/corporal/hook"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"net/http"
	"net/http/httputil"

	"github.com/sirupsen/logrus"
)

type CatchAllHandler struct {
	reverseProxy        *httputil.ReverseProxy
	logger              *logrus.Logger
	userMappingResolver *matrix.UserMappingResolver
	hookRunner          *HookRunner
}

func NewCatchAllHandler(
	reverseProxy *httputil.ReverseProxy,
	logger *logrus.Logger,
	userMappingResolver *matrix.UserMappingResolver,
	hookRunner *HookRunner,
) *CatchAllHandler {
	return &CatchAllHandler{
		reverseProxy:        reverseProxy,
		logger:              logger,
		userMappingResolver: userMappingResolver,
		hookRunner:          hookRunner,
	}
}

func (me *CatchAllHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := me.logger.WithField("method", r.Method)
	logger = logger.WithField("uri", r.RequestURI)

	if r.Method == "OPTIONS" {
		// As per the specification, all servers should be replying to OPTIONS requests identically
		// ( see https://matrix.org/speculator/spec/HEAD/client_server/unstable.html#web-browser-clients ) ,
		// so we might as well do it here and bypass the proxying work.

		logger.Debugf("HTTP gateway: replying to OPTIONS")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// It's useful for hooks to know who the logged-in user is (if any).
	// We try to figure out who it is, but don't fail hard if we can't.
	accessToken := httphelp.GetAccessTokenFromRequest(r)
	isAuthenticated := false
	if accessToken != "" {
		userId, err := me.userMappingResolver.ResolveByAccessToken(accessToken)
		if err == nil {
			isAuthenticated = true
			r = r.WithContext(context.WithValue(r.Context(), "accessToken", accessToken))
			r = r.WithContext(context.WithValue(r.Context(), "userId", userId))
		}
	}

	httpResponseModifierFuncs := make([]hook.HttpResponseModifierFunc, 0)

	// This "runs" both before and after hooks.
	// Before hooks run early on and may abort execution right here.
	// After hooks just schedule HTTP response modifier functions and will actually run later on.
	for _, eventType := range me.orderedEventTypesByAuthStatus(isAuthenticated) {
		if !me.runHook(eventType, w, r, logger, &httpResponseModifierFuncs) {
			return
		}
	}

	reverseProxyToUse := me.reverseProxy

	if len(httpResponseModifierFuncs) == 0 {
		logger.Debugf("HTTP gateway (catch-all): proxying")
	} else {
		logger.Debugf("HTTP gateway (catch-all): proxying (with response modification)")

		reverseProxyCopy := *reverseProxyToUse
		reverseProxyCopy.ModifyResponse = hook.CreateChainedHttpResponseModifierFunc(httpResponseModifierFuncs)
		reverseProxyToUse = &reverseProxyCopy
	}

	reverseProxyToUse.ServeHTTP(w, r)
}

// runHook runs the first matching hook of a given type, possibly injects a response modifier and returns false if we should stop execution
func (me *CatchAllHandler) runHook(
	eventType string,
	w http.ResponseWriter,
	r *http.Request,
	logger *logrus.Entry,
	httpResponseModifierFuncs *[]hook.HttpResponseModifierFunc,
) bool {
	hookResult := me.hookRunner.RunFirstMatchingType(eventType, w, r, logger)
	if hookResult.ResponseSent {
		logger.WithField("hookId", hookResult.Hook.ID).WithField("hookEventType", hookResult.Hook.EventType).Infoln(
			"HTTP gateway (catch-all): hook delivered a response, so we're not proceeding further",
		)
		return false
	}

	if hookResult.ReverseProxyResponseModifier != nil {
		*httpResponseModifierFuncs = append(*httpResponseModifierFuncs, *hookResult.ReverseProxyResponseModifier)
	}

	return true
}

// orderedEventTypesByAuthStatus returns an ordered list of hook event types as they should be executed.
// Before hooks first, followed by after hooks.
//
// Before & after hooks get bundled together, but we execute/initialize them all at once.
func (me *CatchAllHandler) orderedEventTypesByAuthStatus(isAuthenticated bool) []string {
	hooksToRun := []string{hook.EventTypeBeforeAnyRequest}

	if isAuthenticated {
		hooksToRun = append(
			hooksToRun,
			hook.EventTypeBeforeAuthenticatedRequest,
			hook.EventTypeAfterAnyRequest,
			hook.EventTypeAfterAuthenticatedRequest,
		)
	} else {
		hooksToRun = append(
			hooksToRun,
			hook.EventTypeBeforeUnauthenticatedRequest,
			hook.EventTypeAfterAnyRequest,
			hook.EventTypeAfterUnauthenticatedRequest,
		)
	}

	return hooksToRun
}
