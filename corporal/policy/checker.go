package policy

import (
	"devture-matrix-corporal/corporal/util"
)

type Checker struct {
}

func NewChecker() *Checker {
	return &Checker{}
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
