package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"

	"github.com/gorilla/mux"
)

// CheckProfileSetDisplayName is a policy checker for: /_matrix/client/{apiVersion:(r0|v3)}/profile/{targetUserId}/displayname
func CheckProfileSetDisplayName(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	targetUserId := mux.Vars(r)["targetUserId"]

	if userId != targetUserId {
		// Trying to set a display name for someone else. We don't care about this.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		// Not a user we manage.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	if !checker.CanUserUseCustomDisplayName(policy, userId) {
		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorForbidden,
			ErrorMessage: "Denied by policy",
		}
	}

	var payload matrix.ApiUserProfileDisplayNameRequestPayload
	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorBadJson,
			ErrorMessage: err.Error(),
		}
	}

	if matrix.IsUserDeactivatedAccordingToDisplayName(payload.DisplayName) {
		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorForbidden,
			ErrorMessage: "Denied - unallowed display name",
		}
	}

	return PolicyCheckResponse{
		Allow: true,
	}
}

// CheckProfileSetAvatarUrl is a policy checker for: /_matrix/client/{apiVersion:(r0|v3)}/profile/{targetUserId}/avatar_url
func CheckProfileSetAvatarUrl(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	targetUserId := mux.Vars(r)["targetUserId"]

	if userId != targetUserId {
		// Trying to set a display name for someone else. We don't care about this.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		// Not a user we manage.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	if !checker.CanUserUseCustomAvatar(policy, userId) {
		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorForbidden,
			ErrorMessage: "Denied by policy",
		}
	}

	return PolicyCheckResponse{
		Allow: true,
	}
}
