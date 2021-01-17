package handler

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/userauth"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type internalRestAuthHandler struct {
	policyStore          *policy.Store
	homeserverDomainName string
	configuration        configuration.HttpGatewayInternalRESTAuth
	userAuthChecker      *userauth.Checker
	logger               *logrus.Logger

	whitelistedIPBlocks *[]*net.IPNet
}

func NewInternalRESTAuthHandler(
	policyStore *policy.Store,
	homeserverDomainName string,
	configuration configuration.HttpGatewayInternalRESTAuth,
	userAuthChecker *userauth.Checker,
	logger *logrus.Logger,
) *internalRestAuthHandler {
	whitelistedIPBlocks, err := determineWhitelistedIPBlocks(configuration, logger)
	if err != nil {
		logger.Panic(fmt.Errorf("Failed parsing IPNetworkWhitelist: %s", err))
	}

	return &internalRestAuthHandler{
		policyStore:          policyStore,
		homeserverDomainName: homeserverDomainName,
		configuration:        configuration,
		userAuthChecker:      userAuthChecker,
		logger:               logger,

		whitelistedIPBlocks: whitelistedIPBlocks,
	}
}

func (me *internalRestAuthHandler) RegisterRoutesWithRouter(router *mux.Router) {
	router.HandleFunc("/_matrix/corporal/_matrix-internal/identity/v1/check_credentials", me.actionCheckCredentials).Methods("POST")
}

func (me *internalRestAuthHandler) actionCheckCredentials(w http.ResponseWriter, r *http.Request) {
	if me.configuration.Enabled == nil || (*me.configuration.Enabled) == false {
		httphelp.RespondWithMatrixError(w, http.StatusForbidden, matrix.ErrorForbidden, "Internal REST auth is not enabled")
		return
	}

	logger := me.logger.WithField("method", r.Method)
	logger = logger.WithField("uri", r.RequestURI)
	logger.Info("HTTP gateway: internal REST authentication")

	err := me.checkIfRequestIsAllowed(r, logger)
	if err != nil {
		me.logger.Debug(err)
		httphelp.RespondWithMatrixError(w, http.StatusForbidden, matrix.ErrorForbidden, "Refusing to authenticate this HTTP request (bad source IP)")
		return
	}

	requestPayload := userauth.NewRestAuthRequest("", "")

	err = httphelp.GetJsonFromRequestBody(r, &requestPayload)
	if err != nil {
		httphelp.RespondWithMatrixError(w, http.StatusBadRequest, matrix.ErrorBadJson, "Bad request payload")
		return
	}

	policyObj := me.policyStore.Get()
	if policyObj == nil {
		httphelp.RespondWithMatrixError(w, http.StatusInternalServerError, matrix.ErrorUnknown, "Missing policy")
		return
	}

	logger = logger.WithField("userId", requestPayload.User.Id)

	userIDFull, err := matrix.DetermineFullUserId(requestPayload.User.Id, me.homeserverDomainName)
	if err != nil {
		logger.Debug("Cannot construct user id")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	// Replace the logging field with a (potentially) better one
	logger = logger.WithField("userId", userIDFull)

	if !matrix.IsFullUserIdOfDomain(userIDFull, me.homeserverDomainName) {
		logger.Debug("Refusing to authenticate foreign users")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	userPolicy := policyObj.GetUserPolicyByUserId(userIDFull)
	if userPolicy == nil {
		logger.Debug("Refusing to authenticate non-managed user")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	if !userPolicy.Active {
		logger.Debug("Refusing to authenticate deactivated user")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	if userPolicy.AuthType == userauth.UserAuthTypePassthrough {
		// UserAuthTypePassthrough is a special AuthType, authentication for which is not meant to be handled by us.
		// Users are created with an initial password as defined in userPolicy.AuthCredential,
		// but password-management is then potentially left to the homeserver (depending on policyObj.Flags.AllowCustomPassthroughUserPasswords).
		// Authentication always happens at the homeserver.
		//
		// Thus, we reject it here.
		logger.Debug("Refusing to authenticate passthrough users")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	// Authentication for all other auth types is handled by us (below)

	logger = logger.WithField("authType", userPolicy.AuthType)

	isAuthenticated, err := me.userAuthChecker.Check(
		userIDFull,
		requestPayload.User.Password,
		userPolicy.AuthType,
		userPolicy.AuthCredential,
	)
	if err != nil {
		logger.Warn(err)
		httphelp.RespondWithMatrixError(w, http.StatusInternalServerError, matrix.ErrorUnknown, "Internal authenticator error")
		return
	}

	if !isAuthenticated {
		logger.Debug("Authentication failed")
		httphelp.RespondWithJSON(w, http.StatusOK, userauth.NewUnsuccessfulRestAuthResponse())
		return
	}

	httphelp.RespondWithJSON(w, http.StatusOK, userauth.RestAuthResponse{
		Auth: userauth.RestAuthResponseAuth{
			Success:  true,
			MatrixID: userIDFull,
			Profile: &userauth.RestAuthResponseAuthProfile{
				DisplayName: userPolicy.DisplayName,
			},
		},
	})
}

func (me *internalRestAuthHandler) checkIfRequestIsAllowed(r *http.Request, logger *logrus.Entry) error {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return err
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return fmt.Errorf("Failed to parse IP: `%s`", ip)
	}

	logger.Debugf("Checking if IP address `%s` is allowed..", userIP)

	if me.whitelistedIPBlocks == nil {
		// No whitelist at all means allowed.
		return nil
	}

	if !isWhitelistedIPAddress(userIP, *me.whitelistedIPBlocks) {
		logger.Debugf("Determined that %s is NOT allowed", userIP)
		return fmt.Errorf("Not allowed from this IP address")
	}

	return nil
}

func determineWhitelistedIPBlocks(configuration configuration.HttpGatewayInternalRESTAuth, logger *logrus.Logger) (*[]*net.IPNet, error) {
	if configuration.Enabled == nil || (*configuration.Enabled) == false {
		// Doing extra work (and logging) below is useless when Internal REST Auth is not even enabled.
		return nil, nil
	}

	if configuration.IPNetworkWhitelist == nil {
		// An undefined list means "all local/private addresses"
		whitelist := []string{
			"127.0.0.0/8",    // IPv4 loopback
			"10.0.0.0/8",     // RFC1918
			"172.16.0.0/12",  // RFC1918
			"192.168.0.0/16", // RFC1918
			"169.254.0.0/16", // RFC3927 link-local
			"::1/128",        // IPv6 loopback
			"fe80::/10",      // IPv6 link-local
			"fc00::/7",       // IPv6 unique local addr
		}

		return cidrListToBlockList(whitelist)
	}

	// An explicitly defined empty list means "allow everything".
	if len(*configuration.IPNetworkWhitelist) == 0 {
		return nil, nil
	}

	// And finally, a list with at least some entries will get utilized.
	logger.Info("HTTP Internal REST Auth will only be accessible from: %s", *configuration.IPNetworkWhitelist)

	return cidrListToBlockList(*configuration.IPNetworkWhitelist)
}

// Adapted from: https://stackoverflow.com/a/50825191
func cidrListToBlockList(cidrList []string) (*[]*net.IPNet, error) {
	var blocks []*net.IPNet

	for _, cidr := range cidrList {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("Failed parsing %q: %v", cidr, err)
		}

		blocks = append(blocks, block)
	}

	return &blocks, nil
}

// Adapted from: https://stackoverflow.com/a/50825191
func isWhitelistedIPAddress(ip net.IP, allowedNetworks []*net.IPNet) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range allowedNetworks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// Ensure interface is implemented
var _ httphelp.HandlerRegistrator = &internalRestAuthHandler{}
