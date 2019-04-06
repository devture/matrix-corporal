package policycheck

import (
	"context"
	"devture-matrix-corporal/corporal/httphelp"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"net/http"

	"github.com/gorilla/mux"
)

// CheckRoomLeave is a policy checker for: /_matrix/client/r0/rooms/{roomId}/leave
func CheckRoomLeave(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	roomId := mux.Vars(r)["roomId"]

	if !checker.CanUserLeaveRoom(policy, userId, roomId) {
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

// CheckRoomMembershipStateChange is a policy checker for: /_matrix/client/r0/rooms/{roomId}/state/m.room.member/{memberId}
func CheckRoomMembershipStateChange(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	roomId := mux.Vars(r)["roomId"]
	memberId := mux.Vars(r)["memberId"]

	if userId != memberId {
		// Someone is trying to update the membership details of another member.
		// Let it go through and let the upstream server's policies apply, whatever they may be.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	// Someone is trying to modify their own membership state.
	//
	// This may be an attempt to leave (or may even possibly join?) the room,
	// but it may also be an attempt to change one's in-room avatar or name.
	//
	// Let's forbid all of these.
	if !checker.CanUserChangeOwnMembershipStateInRoom(policy, userId, roomId) {
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

// CheckRoomKick is a policy checker for: /_matrix/client/r0/rooms/{roomId}/kick
func CheckRoomKick(r *http.Request, ctx context.Context, policy policy.Policy, checker policy.Checker) PolicyCheckResponse {
	userId := ctx.Value("userId").(string)
	roomId := mux.Vars(r)["roomId"]

	if checker.CanUserChangeOwnMembershipStateInRoom(policy, userId, roomId) {
		// As an optimization, leave early if the current user can change own membership state.
		//
		// We still don't know if the current user tries to kick himself or someone else,
		// but we don't care much either here.
		//
		// If kicking self, we've just checked that it's allowed.
		// If kicking another, we let it go through, so that the upstream server's policies apply.
		return PolicyCheckResponse{
			Allow: true,
		}
	}

	var payload matrix.ApiRoomKickRequestPayload
	err := httphelp.GetJsonFromRequestBody(r, &payload)
	if err != nil {
		return PolicyCheckResponse{
			Allow:        false,
			ErrorCode:    matrix.ErrorBadJson,
			ErrorMessage: err.Error(),
		}
	}

	if userId == payload.UserID {
		// We already confirmed that the current user cannot kick self (see above),
		// so we can outright reject this now.
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
