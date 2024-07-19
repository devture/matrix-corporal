package computator

import (
	"devture-matrix-corporal/corporal/avatar"
	"devture-matrix-corporal/corporal/connector"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/reconciliation"
	"devture-matrix-corporal/corporal/userauth"
	"devture-matrix-corporal/corporal/util"
	"fmt"

	"github.com/sirupsen/logrus"
)

type ReconciliationStateComputator struct {
	logger *logrus.Logger
}

func NewReconciliationStateComputator(logger *logrus.Logger) *ReconciliationStateComputator {
	return &ReconciliationStateComputator{
		logger: logger,
	}
}

func (me *ReconciliationStateComputator) Compute(
	currentState *connector.CurrentState,
	policy *policy.Policy,
) (*reconciliation.State, error) {
	reconciliationState := &reconciliation.State{
		Actions: make([]*reconciliation.StateAction, 0),
	}

	computedActions := make([]*reconciliation.StateAction, 0)
	for _, userPolicy := range policy.User {
		userId := userPolicy.Id

		currentUserStateOrNil := currentState.GetUserStateByUserId(userId)

		actions := me.computeUserChanges(
			userId,
			currentUserStateOrNil,
			policy,
			userPolicy,
		)

		computedActions = append(computedActions, actions...)
	}

	// Group all set power actions for each room
	groupedActions := make([]*reconciliation.StateAction, 0)

	setPowerActionsByRoomId := make(map[string]*reconciliation.StateAction)

	for _, action := range computedActions {
		if action.Type != reconciliation.ActionRoomUserSetPowerLevel {
			groupedActions = append(groupedActions, action)
			continue
		}

		roomId, err := action.GetStringPayloadDataByKey("roomId")
		if err != nil {
			return nil, err
		}

		powerLevel, err := action.GetIntPayloadDataByKey("powerLevel")
		if err != nil {
			return nil, err
		}

		userId, err := action.GetStringPayloadDataByKey("userId")
		if err != nil {
			return nil, err
		}

		setPowerAction, exists := setPowerActionsByRoomId[roomId]
		if !exists {
			userPowerByRoom := make(map[string]int)
			userPowerByRoom[userId] = powerLevel
			newAction := &reconciliation.StateAction{
				Type: reconciliation.ActionRoomUsersSetPowerLevels,
				Payload: map[string]interface{}{
					"roomPowerForUserId": userPowerByRoom,
					"roomId":             roomId,
				},
			}
			setPowerActionsByRoomId[roomId] = newAction
		} else {
			roomPowerPayload := setPowerAction.Payload["roomPowerForUserId"].(map[string]int)
			roomPowerPayload[userId] = powerLevel
		}
	}

	for _, action := range setPowerActionsByRoomId {
		groupedActions = append(groupedActions, action)
	}
	reconciliationState.Actions = groupedActions

	return reconciliationState, nil
}

func (me *ReconciliationStateComputator) computeUserChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	policy *policy.Policy,
	userPolicy *policy.UserPolicy,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	actions = append(
		actions,
		me.computeUserActivationChanges(userId, currentUserState, policy, userPolicy)...,
	)

	if !userPolicy.Active {
		// Accounts that have never been active or are being deactivated now,
		// should not go through the other changes that appear below.
		return actions
	}

	actions = append(
		actions,
		me.computeUserProfileDataChanges(userId, currentUserState, policy, userPolicy)...,
	)

	actions = append(
		actions,
		me.computeUserMembershipChanges(userId, currentUserState, userPolicy, policy.ManagedRoomIds)...,
	)

	return actions
}

func (me *ReconciliationStateComputator) computeUserActivationChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	policy *policy.Policy,
	userPolicy *policy.UserPolicy,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	if currentUserState == nil {
		if userPolicy.Active {
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionUserCreate,
				Payload: map[string]interface{}{
					"userId":   userPolicy.Id,
					"password": me.generateInitialPasswordForUser(*userPolicy),
				},
			})
		}

		return actions
	}

	if !userPolicy.Active {
		// If the user is supposed to be inactive,
		// we want to ensure that it has left all rooms first,
		// before possibly proceeding with a deactivation process.
		actions = append(
			actions,
			me.computeUserMembershipChanges(userId, currentUserState, userPolicy, policy.ManagedRoomIds)...,
		)
	}

	if currentUserState.Active {
		if !userPolicy.Active {
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionUserDeactivate,
				Payload: map[string]interface{}{
					"userId": userPolicy.Id,
				},
			})
		}
	} else {
		if userPolicy.Active {
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionUserActivate,
				Payload: map[string]interface{}{
					"userId": userPolicy.Id,
				},
			})
		}
	}

	return actions
}

func (me *ReconciliationStateComputator) computeUserProfileDataChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	policy *policy.Policy,
	userPolicy *policy.UserPolicy,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	actions = append(
		actions,
		me.computeUserProfileDisplayNameChanges(userId, currentUserState, policy, userPolicy)...,
	)

	actions = append(
		actions,
		me.computeUserProfileAvatarChanges(userId, currentUserState, policy, userPolicy)...,
	)

	return actions
}

func (me *ReconciliationStateComputator) computeUserProfileDisplayNameChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	policy *policy.Policy,
	userPolicy *policy.UserPolicy,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	shouldSetDisplayName := false
	if currentUserState == nil {
		if userPolicy.DisplayName != "" {
			// Newly-created users should get their name set to whatever's in the policy
			// (regardless if custom names are allowed or not).
			shouldSetDisplayName = true
		}
	} else {
		if policy.Flags.AllowCustomUserDisplayNames {
			if currentUserState.DisplayName == "" && userPolicy.DisplayName != "" {
				// Even if we allow custom names, we still want to avoid
				// people having empty names.
				// If we have something to set it to, that is..
				shouldSetDisplayName = true
			}
		} else {
			if currentUserState.DisplayName != userPolicy.DisplayName {
				// Existing users may be locked into a specific display name,
				// given that there's a policy flag that requires that.
				shouldSetDisplayName = true
			}
		}
	}

	if shouldSetDisplayName {
		actions = append(actions, &reconciliation.StateAction{
			Type: reconciliation.ActionUserSetDisplayName,
			Payload: map[string]interface{}{
				"userId":      userPolicy.Id,
				"displayName": userPolicy.DisplayName,
			},
		})
	}

	return actions
}

func (me *ReconciliationStateComputator) computeUserProfileAvatarChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	policy *policy.Policy,
	userPolicy *policy.UserPolicy,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	shouldSetAvatar := false
	if currentUserState == nil {
		if userPolicy.AvatarUri != "" {
			// Newly-created users should get their avatar set to whatever's in the policy
			// (regardless if custom avatars are allowed or not).
			shouldSetAvatar = true
		}
	} else {
		if policy.Flags.AllowCustomUserAvatars {
			if currentUserState.AvatarSourceUriHash == avatar.UriHash("") && userPolicy.AvatarUri != "" {
				// Even if we allow custom avatars, we still want to avoid
				// people having empty avatars.
				// If we have something to set it to, that is..
				shouldSetAvatar = true
			}
		} else {
			// Existing users may be locked into a specific avatar,
			// given that there's a policy flag that requires that.
			if currentUserState.AvatarSourceUriHash != avatar.UriHash(userPolicy.AvatarUri) {
				shouldSetAvatar = true
			}
		}
	}

	if shouldSetAvatar {
		actions = append(actions, &reconciliation.StateAction{
			Type: reconciliation.ActionUserSetAvatar,
			Payload: map[string]interface{}{
				"userId":    userPolicy.Id,
				"avatarUri": userPolicy.AvatarUri,
			},
		})
	}

	return actions
}

func (me *ReconciliationStateComputator) computeUserMembershipChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	userPolicy *policy.UserPolicy,
	managedRoomIds []string,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	actions = append(
		actions,
		me.computeUserRoomChanges(userId, currentUserState, userPolicy, managedRoomIds)...,
	)

	return actions
}

func (me *ReconciliationStateComputator) computeUserRoomChanges(
	userId string,
	currentUserState *connector.CurrentUserState,
	userPolicy *policy.UserPolicy,
	managedRoomIds []string,
) []*reconciliation.StateAction {
	var actions []*reconciliation.StateAction

	for _, room := range userPolicy.JoinedRooms {
		if !util.IsStringInArray(room.RoomId, managedRoomIds) {
			me.logger.Warnf(
				"User %s is supposed to be joined to the %s room, but that room is not managed",
				userPolicy.Id,
				room.RoomId,
			)
			continue
		}

		var currentRoomPowerFound *int = nil
		if currentUserState != nil {
			for _, currentRoom := range currentUserState.JoinedRooms {
				if currentRoom.RoomId == room.RoomId {
					currentRoomPowerFound = &currentRoom.PowerLevel
					break
				}
			}
		}

		if currentRoomPowerFound == nil {
			// If the user is not a member of the room, join it
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionRoomJoin,
				Payload: map[string]interface{}{
					"userId": userId,
					"roomId": room.RoomId,
				},
			})
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionRoomUserSetPowerLevel,
				Payload: map[string]interface{}{
					"userId":     userId,
					"roomId":     room.RoomId,
					"powerLevel": room.PowerLevel,
				},
			})

		} else if *currentRoomPowerFound != room.PowerLevel {
			// If the user is a member of the room but has a different power level, change it

			userPowerByRoom := make(map[string]int)
			userPowerByRoom[userId] = room.PowerLevel
			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionRoomUserSetPowerLevel,
				Payload: map[string]interface{}{
					"userId":     userId,
					"roomId":     room.RoomId,
					"powerLevel": room.PowerLevel,
				},
			})
		}
	}

	if currentUserState != nil {
		for _, room := range currentUserState.JoinedRooms {
			if !util.IsStringInArray(room.RoomId, managedRoomIds) {
				//We rightfully ignore rooms we don't care about.
				continue
			}

			isInPolicyJoinedRooms := false
			for _, policyRoom := range userPolicy.JoinedRooms {
				if room.RoomId == policyRoom.RoomId {
					isInPolicyJoinedRooms = true
					break
				}
			}

			if isInPolicyJoinedRooms {
				continue
			}

			actions = append(actions, &reconciliation.StateAction{
				Type: reconciliation.ActionRoomLeave,
				Payload: map[string]interface{}{
					"userId": userId,
					"roomId": room.RoomId,
				},
			})
		}
	}

	return actions
}

func (me *ReconciliationStateComputator) generateInitialPasswordForUser(userPolicy policy.UserPolicy) string {
	// UserAuthTypePassthrough is a special AuthType. Users are created with an initial password as specified in the policy.
	// For such users, authentication is delegated to the homeserver.
	// We can do password matching on our side as well (at least initially), but delegating authentication to the homeserver,
	// allows users to change their password there, etc.
	// The actual password on the homeserver may change over time.
	if userPolicy.AuthType == userauth.UserAuthTypePassthrough {
		return userPolicy.AuthCredential
	}

	// Some other auth type. We create such users with a random password.
	// These passwords are never meant to be given out or used.
	//
	// Whenever we need to authenticate, we can just obtain an access token
	// thanks to shared-secret-auth, regardless of the actual password that the user has been created with on the homeserver.
	// (see ObtainNewAccessTokenForUserId)
	//
	// Whenever users need to log in, we intercept the /login API
	// and possibly turn the call into a request that shared-secret-auth understands
	// (see LoginInterceptor).

	passwordBytes, err := util.GenerateRandomBytes(64)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", passwordBytes)
}
