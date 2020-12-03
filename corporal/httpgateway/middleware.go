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
	regexApiVersionFromUri = regexp.MustCompile("/_matrix/client/r([^/]+)")

	supportedApiVersions = []string{
		//We only support r0 for the time being.
		"0",
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

		releaseVersion := matches[1] // Something like `0`

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
