package httpapi

import (
	"context"
	"crypto/subtle"
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/httpapi/handler"
	"devture-matrix-corporal/corporal/httphelp"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	logger              *logrus.Logger
	configuration       configuration.HttpApi
	handlerRegistrators []httphelp.HandlerRegistrator
	writeTimeout        time.Duration

	server *http.Server
}

func NewServer(
	logger *logrus.Logger,
	configuration configuration.HttpApi,
	handlerRegistrators []httphelp.HandlerRegistrator,
	writeTimeout time.Duration,
) *Server {
	return &Server{
		logger:              logger,
		configuration:       configuration,
		handlerRegistrators: handlerRegistrators,
		writeTimeout:        writeTimeout,

		server: nil,
	}
}

func (me *Server) Start() error {
	me.server = &http.Server{
		Handler:      me.createRouter(),
		Addr:         me.configuration.ListenAddress,
		WriteTimeout: me.writeTimeout,
		ReadTimeout:  15 * time.Second,
	}

	me.logger.Infof("Starting HTTP API Server on %s", me.server.Addr)

	go func() {
		err := me.server.ListenAndServe()
		if err != http.ErrServerClosed {
			me.logger.Panicf("HTTP API Server error: %s", err)
		}
	}()

	return nil
}

func (me *Server) Stop() error {
	if me.server == nil {
		return nil
	}

	me.logger.Infoln("Stopping HTTP API Server")
	me.server.Shutdown(context.Background())

	return nil
}

func (me *Server) createRouter() http.Handler {
	r := mux.NewRouter()

	r.Use(me.denyUnauthorizedAccessMiddleware)

	r.Use(me.loggingMiddleware)

	for _, registrator := range me.handlerRegistrators {
		registrator.RegisterRoutesWithRouter(r)
	}

	return r
}

func (me *Server) denyUnauthorizedAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)

		accessToken := httphelp.GetAccessTokenFromRequest(r)
		if accessToken == "" {
			logger.Infof("HTTP API: rejecting (missing access token)")

			handler.Respond(w, http.StatusUnauthorized, handler.ApiResponseError{
				ErrorCode:    handler.ErrorCodeMissingToken,
				ErrorMessage: "Missing access token",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(accessToken), []byte(me.configuration.AuthorizationBearerToken)) != 1 {
			logger.Infof("HTTP API: rejecting (bad access token)")

			handler.Respond(w, http.StatusUnauthorized, handler.ApiResponseError{
				ErrorCode:    handler.ErrorCodeUnknownToken,
				ErrorMessage: "Bad access token",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (me *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)

		logger.Infoln("HTTP API: handling request")

		next.ServeHTTP(w, r)
	})
}
