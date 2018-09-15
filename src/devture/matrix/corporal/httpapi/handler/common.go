package handler

import (
	"devture/matrix/corporal/matrix"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	ErrorCodeMissingToken     = matrix.ErrorMissingToken
	ErrorCodeUnknownToken     = matrix.ErrorUnknownToken
	ErrorCodeBadJson          = matrix.ErrorBadJson
	ErrorCodeUnknown          = matrix.ErrorUnknown
	ErrorInvalidUsername      = matrix.ErrorInvalidUsername
	ErrorCodeMissingParameter = matrix.ErrorMissingParameter
)

type HandlerRegistrator interface {
	RegisterRoutesWithRouter(router *mux.Router)
}

// ApiResponseError is a "standard error response" as per the Matrix Client-Server specification.
// All Matrix Corporal HTTP API calls that trigger an error return a response like this.
type ApiResponseError struct {
	ErrorCode    string `json:"errcode"`
	ErrorMessage string `json:"error"`
}

func Respond(w http.ResponseWriter, httpStatusCode int, resp interface{}) {
	w.WriteHeader(httpStatusCode)

	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Errorf("Could not create JSON response for: %s", resp))
	}

	w.Write(respBytes)
}
