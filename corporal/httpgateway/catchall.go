package httpgateway

import (
	"devture-matrix-corporal/corporal/hook"
	"net/http"
	"net/http/httputil"

	"github.com/sirupsen/logrus"
)

type CatchAllHandler struct {
	reverseProxy *httputil.ReverseProxy
	logger       *logrus.Logger
	hookRunner   *HookRunner
}

func NewCatchAllHandler(
	reverseProxy *httputil.ReverseProxy,
	logger *logrus.Logger,
	hookRunner *HookRunner,
) *CatchAllHandler {
	return &CatchAllHandler{
		reverseProxy: reverseProxy,
		logger:       logger,
		hookRunner:   hookRunner,
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

	hookResult := me.hookRunner.RunFirstMatchingType(hook.EventTypeBeforeAnyRequest, w, r, logger)
	if hookResult.SkipProceedingFurther {
		logger.Debugf("HTTP gateway (catch-all): %s hook said we should not proceed further", hook.EventTypeBeforeAnyRequest)
		return
	}

	reverseProxyToUse := me.reverseProxy

	if hookResult.ReverseProxyResponseModifier == nil {
		logger.Debugf("HTTP gateway (catch-all): proxying")
	} else {
		logger.Debugf("HTTP gateway (catch-all): proxying (with response modification)")

		reverseProxyCopy := *reverseProxyToUse
		reverseProxyCopy.ModifyResponse = *hookResult.ReverseProxyResponseModifier

		reverseProxyToUse = &reverseProxyCopy
	}

	reverseProxyToUse.ServeHTTP(w, r)
}
