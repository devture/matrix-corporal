package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"

	"github.com/gorilla/mux"
)

// CheckCommunitySelfLeave is a policy checker for: /_matrix/client/{apiVersion:(r0|v3)}/groups/{communityId}/self/leave
func CheckCommunitySelfLeave(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	communityId := mux.Vars(r)["communityId"]

	if !checker.CanUserLeaveCommunity(policy, userId, communityId) {
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
