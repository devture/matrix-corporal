package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/userauth"
	"net/http"
)

// CheckUserDeactivate is a policy checker for: /_matrix/client/{apiVersion:(r0|v3)}/account/deactivate
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

// CheckUserSetPassword is a policy checker for: /_matrix/client/{apiVersion:(r0|v3)}/account/password
func CheckUserSetPassword(r *http.Request, ctx context.Context, policyObj policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userIdOrNil := ctx.Value("userId")
	userId, ok := userIdOrNil.(string)

	if !ok {
		// Unauthenticated request. This is a password-forgotten / password-reset flow.
		//
		// Since this is an unauthetnicated request, we don't really know who it is.
		// The request payload contains an `auth` field with 3pid validation information, etc.,
		// so the upstream server can figure it out, but it's not easy for us to do so.
		//
		// Ideally, we'd be able to map that to a user and perform our regular policy checks
		// (passthrough users being allowed; others not).
		//
		// But right now, we can't. So we either let everyone through or we don't.
		if policyObj.Flags.AllowUnauthenticatedPasswordResets {
			return PolicyCheckResponse{
				Allow: true,
			}
		}

		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorForbidden,
			ErrorMessage: "Denied: unauthenticted password requests are not allowed on this server",
		}
	}

	// Authenticated request. This is most likely a user trying to assign a new password.
	// For unmanaged users, we'll allow it.
	// For managed users, we'll consult with the policy.

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
			ErrorMessage: "Denied: passthrough user, but policy does not allow password changes",
		}
	}

	return PolicyCheckResponse{
		Allow:        false,
		ErrorCode:    matrix.ErrorForbidden,
		ErrorMessage: "Denied: non-passthrough users are always authenticated against matrix-corporal, so password resets make no sense",
	}
}
