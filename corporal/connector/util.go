package connector

import (
	"strings"

	"github.com/matrix-org/gomatrix"
)

func buildPrefixlessURL(client *gomatrix.Client, path string, queryParams map[string]string) string {
	url := client.BuildURLWithQuery([]string{path}, queryParams)

	// The URL-building function above forces us under the `/_matrix/client/r0/` prefix.
	// We'd like to work at the top-level though, hence this hack.
	return strings.Replace(url, "/_matrix/client/r0/", "/", 1)
}
