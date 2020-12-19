package httphelp

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// ResponseBoundHttpWriter implements http.ResponseWriter, but binds data to an http.Response object, instead of writing it anywhere.
//
// This is useful for when we already have a response (e.g. when executing from within a reverse-proxy `.ModifyResponse` handler),
// and we wish to interact with code which relies an http.ResponseWriter.
//
// In such cases, we obviously don't wish to write directly to the socket, but rather modify the existing
// response object, which will then be writen to a socket by something else.
//
// You can use this as a regular `http.ResponseWriter` with the exception that you need to call `.Commit()`
// when you're all done with writing your response payload to it.
// Unlike payload bytes data, headers and status code changes are applied to the response immediately.
type ResponseBoundHttpWriter struct {
	response *http.Response

	responseBytes *bytes.Buffer
}

func NewResponseBoundHttpWriter(response *http.Response) ResponseBoundHttpWriter {
	return ResponseBoundHttpWriter{
		response:      response,
		responseBytes: bytes.NewBuffer([]byte{}),
	}
}

// Header satisfies http.ResponseWriter
func (me ResponseBoundHttpWriter) Header() http.Header {
	return me.response.Header
}

// Write satisfies http.ResponseWriter
func (me ResponseBoundHttpWriter) Write(b []byte) (int, error) {
	return me.responseBytes.Write(b)
}

// WriteHeader satisfies http.ResponseWriter
func (me ResponseBoundHttpWriter) WriteHeader(statusCode int) {
	me.response.StatusCode = statusCode
}

// Commit binds the bytes written using Write() to the response object we're wrapping.
//
// While other things (headers, status code) are bound immediately, the response payload
// is buffered and will only get bound when you call this.
func (me ResponseBoundHttpWriter) Commit() {
	b, err := ioutil.ReadAll(me.responseBytes)
	if err != nil {
		panic(err)
	}

	if len(b) > 0 {
		me.response.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
}

var _ http.ResponseWriter = ResponseBoundHttpWriter{}
