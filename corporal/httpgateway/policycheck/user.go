package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/userauth"
	"net/http"
)

// CheckUserDeactivate is a policy checker for: /_matrix/client/r0/account/deactivate
func CheckUserDeactivate(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)

	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		// Not a user we manage.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	return PolicyCheckResponse{
		Allow:        false,
		ErrorCode:    matrix.ErrorForbidden,
		ErrorMessage: "Denied",
	}
}

// CheckUserSetPassword is a policy checker for: /_matrix/client/r0/account/password
func CheckUserSetPassword(r *http.Request, ctx context.Context, policyObj policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)

	userPolicy := policyObj.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		// Not a user we manage.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	if userPolicy.AuthType == userauth.UserAuthTypePassthrough {
		if policyObj.Flags.AllowCustomPassthroughUserPasswords {
			return PolicyCheckResponse{
				Allow: true,
			}
		}

		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorForbidden,
			ErrorMessage: "Denied: passthrough user, but policy does not allow changes",
		}
	}

	return PolicyCheckResponse{
		Allow:        false,
		ErrorCode:    matrix.ErrorForbidden,
		ErrorMessage: "Denied: non-passthrough users are always authenticated against matrix-corporal",
	}
}
