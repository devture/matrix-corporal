package httphelp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func GetJsonFromRequestBody(r *http.Request, out interface{}) error {
	// Reading an unlimited amount of data from the body is dangerous, but:
	// - we only invoke this after we've authenticated a known user,
	// so at least we're not exposed to the whole world but rather just to our own users
	// - we're not supposed to be the first HTTP server in line,
	// so very large requests would be rejected by the server in front of us
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
