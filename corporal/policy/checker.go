package policy

import (
	"devture-matrix-corporal/corporal/util"
)

type Checker struct {
}

func NewChecker() *Checker {
	return &Checker{}
}

func (me *Checker) CanUserCreateRoom(policy Policy, userId string) bool {
	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy != nil {
		if userPolicy.ForbidRoomCreation != nil {
			return !*userPolicy.ForbidRoomCreation
		}
	}

	// No dedicated policy for this user (likely an unmanaged user) or undefined ForbidRoomCreation policy field.
	// Stick to the global defaults.
	return !policy.Flags.ForbidRoomCreation
}

func (me *Checker) CanUserLeaveRoom(policy Policy, userId string, roomId string) bool {
	return me.CanUserChangeOwnMembershipStateInRoom(policy, userId, roomId)
}

func (me *Checker) CanUserChangeOwnMembershipStateInRoom(policy Policy, userId string, roomId string) bool {
	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		return true
	}

	if util.IsStringInArray(roomId, userPolicy.JoinedRoomIds) {
		return false
	}

	return true
}

func (me *Checker) CanUserLeaveCommunity(policy Policy, userId string, communityId string) bool {
	return me.CanUserChangeOwnMembershipStateInCommunity(policy, userId, communityId)
}

func (me *Checker) CanUserChangeOwnMembershipStateInCommunity(policy Policy, userId string, communityId string) bool {
	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy == nil {
		return true
	}

	if util.IsStringInArray(communityId, userPolicy.JoinedCommunityIds) {
		return false
	}

	return true
}

func (me *Checker) CanUserUseCustomDisplayName(policy Policy, userId string) bool {
	return policy.Flags.AllowCustomUserDisplayNames
}

func (me *Checker) CanUserUseCustomAvatar(policy Policy, userId string) bool {
	return policy.Flags.AllowCustomUserAvatars
}

//Compares the power level of sender and invited members. Allows invite only within their power level and below.

func (me *Checker) CanSendInvite(policy, userId, memberId)  bool {
	memberPolicy := policy.GetUserPolicyByUserId(memberId)	
	userPolicy := policy.GetUserPolicyByUserId(userId)	
	if memberPolicy == nil {
		return true
	}

	if userPolicy == nil {
		return false
	}

	return memberPolicy.PowerLevel <= userPolicy.PowerLevel
}