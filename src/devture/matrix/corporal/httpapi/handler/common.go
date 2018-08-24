package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type HandlerRegistrator interface {
	RegisterRoutesWithRouter(router *mux.Router)
}

type ApiResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

func Respond(w http.ResponseWriter, httpStatusCode int, resp interface{}) {
	w.WriteHeader(httpStatusCode)

	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Errorf("Could not create JSON response for: %s", resp))
	}

	w.Write(respBytes)
}
