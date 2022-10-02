package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"devture-matrix-corporal/corporal/httphelp"
)

type corporalHandler struct {
	logger *logrus.Logger
}

func NewCorporalHandler(
	logger *logrus.Logger,
) *corporalHandler {
	return &corporalHandler{
		logger: logger,
	}
}

func (me *corporalHandler) RegisterRoutesWithRouter(router *mux.Router) {
	// To make it easy to detect if Matrix Corporal is properly fronting the Matrix Client-Server API,
	// we add this custom non-standard route.
	router.HandleFunc("/_matrix/client/corporal", me.actionCorporalIndex).Methods("GET")
}

func (me *corporalHandler) actionCorporalIndex(w http.ResponseWriter, r *http.Request) {
	logger := me.logger.WithField("method", r.Method)
	logger = logger.WithField("uri", r.RequestURI)
	logger.Debugf("HTTP gateway: serving Matrix Corporal info page")

	_, err := w.Write([]byte("Matrix Client-Server API protected by Matrix Corporal"))
	if err != nil {
		log.Printf("failed writing response: %s", err)
	}
}

// Ensure interface is implemented
var _ httphelp.HandlerRegistrator = &corporalHandler{}
