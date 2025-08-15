package httphelp

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gorilla/mux"
)

type HandlerRegistrator interface {
	RegisterRoutesWithRouter(router *mux.Router)
}

func readBytesAndRecreateReader(source io.ReadCloser) ([]byte, io.ReadCloser, error) {
	// Reading an unlimited amount of data might be dangerous.
	sourceBytes, err := io.ReadAll(source)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read bytes from source reader")
	}

	err = source.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot close source reader")
	}

	return sourceBytes, io.NopCloser(bytes.NewReader(sourceBytes)), nil
}
