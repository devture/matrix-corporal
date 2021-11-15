package httpgateway

import (
	"context"
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/httphelp"
	"net/http"
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

	return r
}
