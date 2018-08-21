package httpgateway

import (
	"net/http"
	"net/http/httputil"

	"github.com/sirupsen/logrus"
)

type CatchAllHandler struct {
	reverseProxy *httputil.ReverseProxy
	logger       *logrus.Logger
}

func NewCatchAllHandler(reverseProxy *httputil.ReverseProxy, logger *logrus.Logger) *CatchAllHandler {
	return &CatchAllHandler{
		reverseProxy: reverseProxy,
		logger:       logger,
	}
}

func (me *CatchAllHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := me.logger.WithField("method", r.Method)
	logger = logger.WithField("uri", r.RequestURI)

	if r.Method == "OPTIONS" {
		// As per the specification, all servers should be replying to OPTIONS requests identically
		// ( see https://matrix.org/speculator/spec/HEAD/client_server/unstable.html#web-browser-clients ) ,
		// so we might as well do it here and bypass the proxying work.

		logger.Debugf("Replying to OPTIONS")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	logger.Debugf("Proxying catch-all")
	me.reverseProxy.ServeHTTP(w, r)
}
