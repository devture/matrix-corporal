package httpgateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/matrix-org/gomatrix"
	"github.com/sirupsen/logrus"
)

func respondWithMatrixError(w http.ResponseWriter, httpStatusCode int, errorCode string, errorMessage string) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(httpStatusCode)

	resp := gomatrix.RespError{
		Err:     errorMessage,
		ErrCode: errorCode,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Errorf("Could not create JSON response for: %s", resp))
	}

	w.Write(respBytes)
}

func createInterceptorErrorResponse(loggingContextFields logrus.Fields, errorCode, errorMessage string) InterceptorResponse {
	return InterceptorResponse{
		Result:               InterceptorResultDeny,
		LoggingContextFields: loggingContextFields,
		ErrorCode:            errorCode,
		ErrorMessage:         errorMessage,
	}
}
