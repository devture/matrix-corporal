package httphelp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/matrix-org/gomatrix"
)

func GetResponseBody(r *http.Response) ([]byte, error) {
	// Reading an unlimited amount of data from the body is dangerous, but:
	// - we're not supposed to be the first HTTP server in line,
	// so very large requests would be rejected by the server in front of us
	bodyBytes, newReader, err := readBytesAndRecreateReader(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot read response body payload: %s", err)
	}

	// We read the body, so we ought to restore it.
	r.Body = newReader

	return bodyBytes, nil
}

func GetJsonFromResponseBody(r *http.Response, out interface{}) error {
	bodyBytes, err := GetResponseBody(r)

	err = json.Unmarshal(bodyBytes, out)
	if err != nil {
		return fmt.Errorf("Cannot understand response body payload (not JSON)")
	}

	return nil
}

func RespondWithMatrixError(w http.ResponseWriter, httpStatusCode int, errorCode string, errorMessage string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
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

func RespondWithBytes(w http.ResponseWriter, httpStatusCode int, contentType string, payload []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(httpStatusCode)

	w.Write(payload)
}

func RespondWithJSON(w http.ResponseWriter, httpStatusCode int, responsePayload interface{}) {
	responsePayloadBytes, err := json.Marshal(responsePayload)
	if err != nil {
		panic(fmt.Errorf("Could not create JSON response for: %s", responsePayload))
	}

	RespondWithBytes(w, httpStatusCode, "application/json", responsePayloadBytes)
}
