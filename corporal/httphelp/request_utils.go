package httphelp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	bodyBytes, newReader, err := readBytesAndRecreateReader(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot read request body payload: %s", err)
	}

	// We read the body, so we ought to restore it,
	// so that other things (like reverse-proxying) can read it later.
	r.Body = newReader

	return bodyBytes, nil
}

func GetJsonFromRequestBody(r *http.Request, out interface{}) error {
	bodyBytes, err := GetRequestBody(r)

	err = json.Unmarshal(bodyBytes, out)
	if err != nil {
		return fmt.Errorf("Cannot understand request body payload (not JSON)")
	}

	return nil
}
