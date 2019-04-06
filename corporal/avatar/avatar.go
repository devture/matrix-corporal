package avatar

import (
	"bytes"
	"devture-matrix-corporal/corporal/util"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Avatar struct {
	ContentType   string
	ContentLength int64
	Body          io.ReadCloser
	UriHash       string
}

type AvatarReader struct {
}

func NewAvatarReader() *AvatarReader {
	return &AvatarReader{}
}

func (me *AvatarReader) Read(avatarUri string) (*Avatar, error) {
	avatar := &Avatar{
		UriHash: UriHash(avatarUri),
	}

	if avatarUri == "" {
		avatar.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
		return avatar, nil
	}

	// Example: data:image/jpeg;base64,data-here
	if strings.HasPrefix(avatarUri, "data:") {
		dataContent := avatarUri[len("data:"):]

		semiColon := strings.Index(dataContent, ";")
		if semiColon == -1 {
			return nil, fmt.Errorf("Malformed data URI, cannot find semicolon")
		}
		avatar.ContentType = dataContent[:semiColon]

		commaPos := strings.Index(dataContent, ",")
		if commaPos == -1 {
			return nil, fmt.Errorf("Malformed data URI, cannot find comma")
		}

		dataBase64 := dataContent[commaPos+1:]

		dataBytes, err := base64.StdEncoding.DecodeString(dataBase64)
		if err != nil {
			return nil, fmt.Errorf("Failed to base64-decode data: %s", err)
		}

		avatar.ContentLength = int64(len(dataBytes))
		avatar.Body = ioutil.NopCloser(bytes.NewReader(dataBytes))

		return avatar, nil
	}

	// Everything else is a URL.

	resp, err := http.Get(avatarUri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 response fetching from URL: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed reading HTTP response body: %s", err)
	}

	avatar.ContentType = resp.Header.Get("Content-Type")
	avatar.ContentLength = int64(len(bodyBytes))
	avatar.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))

	return avatar, nil
}

func UriHash(uri string) string {
	return util.Sha512(uri)
}
