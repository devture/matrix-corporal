package httpgateway

import (
	"bytes"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/userauth"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/matrix-org/gomatrix"

	"github.com/sirupsen/logrus"
)

// LoginInterceptor is an HTTP request interceptor that handles the /login API path.
//
// It's goal is to authenticate users described in the policy and let them get an access token
// if credentials match.
//
// Requests by users which are not described in the policy (also called non-managed-users) are
// forwarded as-is to the Matrix API server. They may get authenticated or not (we don't care about them).
//
// Managed users (those in the policy), however, are authenticated with us.
// Authentication can happen differently for each user described the policy,
// depending on the user's `authType` and `authCredential` fields.
//
// In any case, if authentication is successful, a special fake "password" is generated,
// and the request is forwarded to the Matrix server's /login API with that password.
// The remote server is expected to be configured properly, so it would understand and trust
// our fake passwords and grant access.
// Those passwords are verified and trusted through the `matrix-shared-secret-auth` plugin for Synapse
// and are generated to match via SharedSecretAuthPasswordGenerator.
type LoginInterceptor struct {
	policyStore                       *policy.Store
	homeserverDomainName              string
	userAuthChecker                   *userauth.Checker
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator
}

func NewLoginInterceptor(
	policyStore *policy.Store,
	homeserverDomainName string,
	userAuthChecker *userauth.Checker,
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator,
) *LoginInterceptor {
	return &LoginInterceptor{
		policyStore:                       policyStore,
		homeserverDomainName:              homeserverDomainName,
		userAuthChecker:                   userAuthChecker,
		sharedSecretAuthPasswordGenerator: sharedSecretAuthPasswordGenerator,
	}
}

func (me *LoginInterceptor) Intercept(r *http.Request) InterceptorResponse {
	loggingContextFields := logrus.Fields{}

	var payload gomatrix.ReqLogin

	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		loggingContextFields["err"] = err.Error()
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorBadJson, "Bad input")
	}

	loggingContextFields["userId"] = payload.User

	userIdFull, err := matrix.DetermineFullUserId(payload.User, me.homeserverDomainName)
	if err != nil {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Cannot interpret user id")
	}

	// Replace the logging field with a (potentially) better one
	loggingContextFields["userId"] = userIdFull

	if !matrix.IsFullUserIdOfDomain(userIdFull, me.homeserverDomainName) {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Rejecting non-own domains")
	}

	if payload.Type != matrix.LoginTypePassword {
		// This is a Token Authentication or other request.
		// We don't know of any use cases for this,
		// so we'll outright reject it for now, just to be on the safe side.
		message := fmt.Sprintf("Denying type: %s", payload.Type)
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, message)
	}

	policy := me.policyStore.Get()
	if policy == nil {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUnknown, "Missing policy")
	}

	userPolicy := policy.GetUserPolicyByUserId(userIdFull)
	if userPolicy == nil {
		// Not a user we manage.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return InterceptorResponse{
			Result:               InterceptorResultProxy,
			LoggingContextFields: loggingContextFields,
		}
	}

	if !userPolicy.Active {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Deactivated in policy")
	}

	loggingContextFields["authType"] = userPolicy.AuthType

	isAuthenticated, err := me.userAuthChecker.Check(
		userIdFull,
		payload.Password,
		userPolicy.AuthType,
		userPolicy.AuthCredential,
	)
	if err != nil {
		loggingContextFields["err"] = err.Error()
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUnknown, "Internal authenticator error")
	}

	if !isAuthenticated {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Failed authentication")
	}

	// We don't need to do it, but let's ensure the payload uses the full user id.
	payload.User = userIdFull
	payload.Password = me.sharedSecretAuthPasswordGenerator.GenerateForUserId(userIdFull)

	newBodyBytes, err := json.Marshal(payload)
	if err != nil {
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUnknown, "Internal error")
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(newBodyBytes))
	r.ContentLength = int64(len(newBodyBytes))

	return InterceptorResponse{
		Result:               InterceptorResultProxy,
		LoggingContextFields: loggingContextFields,
	}
}
