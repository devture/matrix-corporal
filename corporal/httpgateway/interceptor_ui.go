package httpgateway

import (
	"bytes"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/userauth"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

type UiAuthInterceptor struct {
	policyStore                       *policy.Store
	homeserverDomainName              string
	userAuthChecker                   *userauth.Checker
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator
}

func NewUiAuthInterceptor(
	policyStore *policy.Store,
	homeserverDomainName string,
	userAuthChecker *userauth.Checker,
	sharedSecretAuthPasswordGenerator *matrix.SharedSecretAuthPasswordGenerator,
) *UiAuthInterceptor {
	return &UiAuthInterceptor{
		policyStore:                       policyStore,
		homeserverDomainName:              homeserverDomainName,
		userAuthChecker:                   userAuthChecker,
		sharedSecretAuthPasswordGenerator: sharedSecretAuthPasswordGenerator,
	}
}

func (me *UiAuthInterceptor) Intercept(r *http.Request) InterceptorResponse {
	loggingContextFields := logrus.Fields{}

	var payload map[string]interface{}

	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		loggingContextFields["err"] = err.Error()
		return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorBadJson, "Bad input")
	}

	// This all has to be done manually since UI auth can have custom data anywhere
	if payload["auth"] != nil {
		auth := payload["auth"].(map[string]interface{})
		if auth["type"].(string) == "m.login.password" {
			pass, pass_ok := auth["password"].(string)
			if auth["identifier"] == nil || !pass_ok {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorBadJson, "Bad input")
			}
			ident := auth["identifier"].(map[string]interface{})
			id_type, id_type_ok := ident["type"].(string)
			user, user_ok := ident["user"].(string)
			if !id_type_ok {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorBadJson, "Bad input")
			}
			// TODO: Support 3pids here
			if id_type != "m.id.user" || !user_ok {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUnknown, "Third party user IDs not yet supported")
			}

			loggingContextFields["userId"] = user

			userIdFull, err := matrix.DetermineFullUserId(user, me.homeserverDomainName)
			if err != nil {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Cannot interpret user id")
			}

			// Replace the logging field with a (potentially) better one
			loggingContextFields["userId"] = userIdFull

			if !matrix.IsFullUserIdOfDomain(userIdFull, me.homeserverDomainName) {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorForbidden, "Rejecting non-own domains")
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
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUserDeactivated, "Deactivated in policy")
			}

			loggingContextFields["authType"] = userPolicy.AuthType

			isAuthenticated, err := me.userAuthChecker.Check(
				userIdFull,
				pass,
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
			ident["user"] = userIdFull
			auth["identifier"] = ident
			auth["password"] = me.sharedSecretAuthPasswordGenerator.GenerateForUserId(userIdFull)
			payload["auth"] = auth

			newBodyBytes, err := json.Marshal(payload)
			if err != nil {
				return createInterceptorErrorResponse(loggingContextFields, matrix.ErrorUnknown, "Internal error")
			}

			r.Body = ioutil.NopCloser(bytes.NewReader(newBodyBytes))
			r.ContentLength = int64(len(newBodyBytes))
		}
	}
	return InterceptorResponse{
		Result:               InterceptorResultProxy,
		LoggingContextFields: loggingContextFields,
	}
}
