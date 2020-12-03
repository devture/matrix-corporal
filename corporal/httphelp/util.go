package httphelp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/matrix-org/gomatrix"
)

func GetAccessTokenFromRequest(request *http.Request) string {
	accessToken := GetAccessTokenFromRequestHeader(request)
	if accessToken != "" {
		return accessToken
	}

	return GetAccessTokenFromRequestQuery(request)
}

func GetAccessTokenFromRequestHeader(request *http.Request) string {
	authorization := request.Header.Get("Authorization")
	if authorization == "" {
		return ""
	}

	if !strings.HasPrefix(authorization, "Bearer ") {
		return ""
	}

	return authorization[len("Bearer "):]
}

func GetAccessTokenFromRequestQuery(request *http.Request) string {
	err := request.ParseForm()
	if err != nil {
		return ""
	}

	return request.Form.Get("access_token")
}

func GetRequestBody(r *http.Request) ([]byte, error) {
	// Reading an unlimited amount of data from the body is dangerous, but:
	// - we're not supposed to be the first HTTP server in line,
	// so very large requests would be rejected by the server in front of us
	bodyBytes, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Cannot read body payload")
	}

	// We read the body, so we ought to restore it,
	// so that other things (like reverse-proxying) can read it later.
	r.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))

	return bodyBytes, nil
}

func GetJsonFromRequestBody(r *http.Request, out interface{}) error {
	bodyBytes, err := GetRequestBody(r)

	err = json.Unmarshal(bodyBytes, out)
	if err != nil {
		return fmt.Errorf("Cannot understand body payload (not JSON)")
	}

	return nil
}

func GetJsonFromResponseBody(r *http.Response, out interface{}) error {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return fmt.Errorf("Cannot read body payload")
	}

	// We read the body, so we ought to restore it,
	// so that other things (like reverse-proxying) can read it later.
	r.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))

	err = json.Unmarshal(bodyBytes, out)
	if err != nil {
		return fmt.Errorf("Cannot understand body payload (not JSON)")
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
