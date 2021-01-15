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

func (me *Checker) CanUserCreateEncryptedRoom(policy Policy, userId string) bool {
	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy != nil {
		if userPolicy.ForbidEncryptedRoomCreation != nil {
			return !*userPolicy.ForbidEncryptedRoomCreation
		}
	}

	// No dedicated policy for this user (likely an unmanaged user) or undefined ForbidEncryptedRoomCreation policy field.
	// Stick to the global defaults.
	return !policy.Flags.ForbidEncryptedRoomCreation
}

func (me *Checker) CanUserCreateUnencryptedRoom(policy Policy, userId string) bool {
	userPolicy := policy.GetUserPolicyByUserId(userId)
	if userPolicy != nil {
		if userPolicy.ForbidUnencryptedRoomCreation != nil {
			return !*userPolicy.ForbidUnencryptedRoomCreation
		}
	}

	// No dedicated policy for this user (likely an unmanaged user) or undefined ForbidUnencryptedRoomCreation policy field.
	// Stick to the global defaults.
	return !policy.Flags.ForbidUnencryptedRoomCreation
}

func (me *Checker) CanUserSendEventToRoom(policy Policy, userId string, eventType string, roomId string) bool {
	// Everyone can send everything wherywhere now.
	// We don't have policy rules that affect this.
	//
	// However, people can intercept and control this via hooks.
	return true
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
