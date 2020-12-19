package httphelp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

func readBytesAndRecreateReader(source io.ReadCloser) ([]byte, io.ReadCloser, error) {
	// Reading an unlimited amount of data might be dangerous.
	sourceBytes, err := ioutil.ReadAll(source)
	source.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot read bytes from source reader")
	}

	return sourceBytes, ioutil.NopCloser(bytes.NewReader(sourceBytes)), nil
}
