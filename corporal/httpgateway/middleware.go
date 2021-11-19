package httpgateway

import (
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/util"
	"net/http"
	"regexp"
)

var regexApiVersionFromUri *regexp.Regexp

var supportedApiVersions []string

func init() {
	// We'd like to match things like:
	// - `/_matrix/client/r0`
	// - `/_matrix/client/v3` (and other v-prefixed versions in the future)
	// but not match things like: `/_matrix/client/versions`
	regexApiVersionFromUri = regexp.MustCompile(`/_matrix/client/((?:r|v)\d+)`)

	supportedApiVersions = []string{
		"r0",
		"v3",
	}
}

func denyUnsupportedApiVersionsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matches := regexApiVersionFromUri.FindStringSubmatch(r.RequestURI)
		if matches == nil {
			// Some other request. We don't care about it.
			next.ServeHTTP(w, r)
			return
		}

		releaseVersion := matches[1] // Something like `r0` or `v3`, etc.

		if util.IsStringInArray(releaseVersion, supportedApiVersions) {
			// We do support this version and can safely let our gateway
			// capture requests for it, etc.
			next.ServeHTTP(w, r)
			return
		}

		// An attempt to call an API that we don't support
		// (that is, that we don't capture and handle below).
		// Letting these requests go through would be a security risk.

		httphelp.RespondWithMatrixError(w, http.StatusForbidden, matrix.ErrorForbidden, "API version not supported by gateway")
	})
}
