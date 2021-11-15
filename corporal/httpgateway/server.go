package httpgateway

import (
	"context"
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/httphelp"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	logger              *logrus.Logger
	configuration       configuration.HttpGateway
	handlerRegistrators []httphelp.HandlerRegistrator
	writeTimeout        time.Duration

	server *http.Server
}

func NewServer(
	logger *logrus.Logger,
	configuration configuration.HttpGateway,
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

	for _, registrator := range me.handlerRegistrators {
		registrator.RegisterRoutesWithRouter(r)
	}

	return me.stripTrailingSlashMiddleware(r)
}

// stripTrailingSlashMiddleware removes trailing slashes from incoming requests to make our routes match.
//
// It's important to us that policy-checked routes are matched, so we can do guarantee policy checking and potentially reject requests.
// Without this slash-stripping middleware, a request for `/_matrix/client/r0/rooms/{roomId}/state/m.room.encryption/` (note the trailing slash)
// does not match our `/_matrix/client/r0/rooms/{roomId}/state/m.room.encryption` policy-checked handler,
// slips through and gets happily served by the homserver.
//
// There's a mux Router `StripSlash(true)` setting, but it doesn't seem to be effective,
// as described here: https://natedenlinger.com/dealing-with-trailing-slashes-on-requesturi-in-go-with-mux/
// We need this middleware as a workaround. It needs be to wrap the router, so it runs early enough.
// If we were to run it as a mux Middleware, it would be too late.
func (me *Server) stripTrailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")

		next.ServeHTTP(w, r)
	})
}
