package httpapi

import (
	"context"
	"crypto/subtle"
	"devture/matrix/corporal/configuration"
	"devture/matrix/corporal/httphelp"
	"devture/matrix/corporal/policy"
	"devture/matrix/corporal/policy/provider"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type apiResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

type Server struct {
	logger         *logrus.Logger
	configuration  configuration.HttpApi
	policyProvider provider.Provider
	policyStore    *policy.Store

	server *http.Server
}

func NewServer(
	logger *logrus.Logger,
	configuration configuration.HttpApi,
	policyProvider provider.Provider,
	policyStore *policy.Store,
) *Server {
	return &Server{
		logger:         logger,
		configuration:  configuration,
		policyProvider: policyProvider,
		policyStore:    policyStore,

		server: nil,
	}
}

func (me *Server) Start() error {
	me.server = &http.Server{
		Handler:      me.createRouter(),
		Addr:         me.configuration.ListenAddress,
		WriteTimeout: 15 * time.Second,
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

	r.HandleFunc(
		"/_matrix/corporal/policy",
		func(w http.ResponseWriter, r *http.Request) {
			var policy policy.Policy

			err := httphelp.GetJsonFromRequestBody(r, &policy)
			if err != nil {
				respond(
					w,
					http.StatusBadRequest,
					apiResponse{
						Ok:    false,
						Error: "Bad body payload",
					},
				)
				return
			}

			err = me.policyStore.Set(&policy)
			if err != nil {
				respond(
					w,
					http.StatusBadRequest,
					apiResponse{
						Ok:    false,
						Error: fmt.Sprintf("Failed to set policy: %s", err),
					},
				)
				return
			}

			respond(w, http.StatusOK, apiResponse{
				Ok: true,
			})
		},
	).Methods("PUT")

	r.HandleFunc(
		"/_matrix/corporal/policy/provider/reload",
		func(w http.ResponseWriter, r *http.Request) {
			go me.policyProvider.Reload()

			respond(w, http.StatusOK, apiResponse{
				Ok: true,
			})
		},
	).Methods("POST")

	return r
}

func (me *Server) denyUnauthorizedAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := me.logger.WithField("method", r.Method)
		logger = logger.WithField("uri", r.RequestURI)

		accessToken := httphelp.GetAccessTokenFromRequest(r)
		if accessToken == "" {
			logger.Debugf("Rejecting (missing access token)")

			respond(
				w,
				http.StatusUnauthorized,
				apiResponse{
					Ok:    false,
					Error: "Missing access token",
				},
			)
			return
		}

		if subtle.ConstantTimeCompare([]byte(accessToken), []byte(me.configuration.AuthorizationBearerToken)) != 1 {
			respond(
				w,
				http.StatusUnauthorized,
				apiResponse{
					Ok:    false,
					Error: "Bad access token",
				},
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, httpStatusCode int, resp apiResponse) {
	w.WriteHeader(httpStatusCode)

	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Errorf("Could not create JSON response for: %s", resp))
	}

	w.Write(respBytes)
}
