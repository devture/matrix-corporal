package reconciler

import (
	"devture-matrix-corporal/corporal/avatar"
	"devture-matrix-corporal/corporal/connector"
	"devture-matrix-corporal/corporal/matrix"
	"devture-matrix-corporal/corporal/policy"
	"devture-matrix-corporal/corporal/reconciliation"
	"devture-matrix-corporal/corporal/reconciliation/computator"
	"fmt"

	"github.com/sirupsen/logrus"
)

const (
	deviceIdReconciler = "Matrix-Corporal-Reconciler"
)

type ReconciliationHandlerFunc func(*connector.AccessTokenContext, *reconciliation.StateAction) error

type Reconciler struct {
	logger              *logrus.Logger
	connector           connector.MatrixConnector
	computator          *computator.ReconciliationStateComputator
	reconciliatorUserId string
	avatarReader        *avatar.AvatarReader

	handlers map[string]ReconciliationHandlerFunc
}

func New(
	logger *logrus.Logger,
	connector connector.MatrixConnector,
	computator *computator.ReconciliationStateComputator,
	reconciliatorUserId string,
	avatarReader *avatar.AvatarReader,
) *Reconciler {
	me := &Reconciler{
		logger:              logger,
		connector:           connector,
		computator:          computator,
		reconciliatorUserId: reconciliatorUserId,
		avatarReader:        avatarReader,
	}

	me.handlers = map[string]ReconciliationHandlerFunc{
		reconciliation.ActionUserCreate:         me.reconcileForActionUserCreate,
		reconciliation.ActionUserSetDisplayName: me.reconcileForActionUserSetDisplayName,
		reconciliation.ActionUserSetAvatar:      me.reconcileForActionUserSetAvatar,
		reconciliation.ActionUserActivate:       me.reconcileForActionUserActivate,
		reconciliation.ActionUserDeactivate:     me.reconcileForActionUserDeactivate,

		reconciliation.ActionCommunityJoin:  me.reconcileForActionCommunityJoin,
		reconciliation.ActionCommunityLeave: me.reconcileForActionCommunityLeave,

		reconciliation.ActionRoomJoin:  me.reconcileForActionRoomJoin,
		reconciliation.ActionRoomLeave: me.reconcileForActionRoomLeave,
	}

	return me
}

func (me *Reconciler) Reconcile(policy *policy.Policy) error {
	// We clean up tokens after ourselves, but it's good to specify some validity anyway.
	// Even if reconciliation takes longer than the validity, it likely wouldn't be a problem,
	// because the token context checks validity times and gives us a fresh token if it encounters an expired one.
	//
	// Still, it's good to use a larger validity time to avoid obtaining too many tokens.
	tokenValiditySeconds := 12 * 60

	ctx := connector.NewAccessTokenContext(me.connector, deviceIdReconciler, tokenValiditySeconds)
	defer ctx.Release()

	currentState, err := me.connector.DetermineCurrentState(ctx, policy.GetManagedUserIds(), me.reconciliatorUserId)
	if err != nil {
		return fmt.Errorf("Failure determining current state: %s", err)
	}

	reconciliationState, err := me.computator.Compute(currentState, policy)
	if err != nil {
		return err
	}

	for _, action := range reconciliationState.Actions {
		logger := me.logger.WithField("action", action.Type)
		logger = logger.WithFields(logrus.Fields(action.Payload))

		handlerFunc, exists := me.handlers[action.Type]
		if !exists {
			err = fmt.Errorf("Missing reconciliation handler")
			logger.Errorf(err.Error())
			return err
		}

		err = handlerFunc(ctx, action)
		if err != nil {
			err = fmt.Errorf("Failed reconciliation handler: %s", err)
			logger.Errorf(err.Error())
			return err
		}

		logger.Infof("Completed reconciliation handler")
	}

	return nil
}

func (me *Reconciler) reconcileForActionUserCreate(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	password, err := action.GetStringPayloadDataByKey("password")
	if err != nil {
		return err
	}

	err = me.connector.EnsureUserAccountExists(userId, password)
	if err != nil {
		return fmt.Errorf("Failed ensuring %s is created: %s", userId, err)
	}

	return nil
}

func (me *Reconciler) reconcileForActionUserSetDisplayName(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	displayName, err := action.GetStringPayloadDataByKey("displayName")
	if err != nil {
		return err
	}

	err = me.connector.SetUserDisplayName(ctx, userId, displayName)
	if err != nil {
		return fmt.Errorf("Failed setting user display name (%s) for %s: %s", displayName, userId, err)
	}

	return nil
}

func (me *Reconciler) reconcileForActionUserSetAvatar(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	avatarUri, err := action.GetStringPayloadDataByKey("avatarUri")
	if err != nil {
		return err
	}

	avatar, err := me.avatarReader.Read(avatarUri)
	if err != nil {
		return fmt.Errorf("Failed reading user avatar from %s: %s", avatarUri, err)
	}

	err = me.connector.SetUserAvatar(ctx, userId, avatar)
	if err != nil {
		return fmt.Errorf("Failed setting user avatar for %s: %s", userId, err)
	}

	return nil
}

func (me *Reconciler) reconcileForActionUserActivate(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	userProfile, err := me.connector.GetUserProfileByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("Failed retrieving user profile: %s", err)
	}

	if !matrix.IsUserDeactivatedAccordingToDisplayName(userProfile.DisplayName) {
		// Already done. Nothing to do.
		return nil
	}

	newDisplayName := matrix.CleanDeactivationMarkerFromDisplayName(userProfile.DisplayName)

	err = me.connector.SetUserDisplayName(ctx, userId, newDisplayName)
	if err != nil {
		return fmt.Errorf("Failed setting display name (%s) for %s: %s", newDisplayName, userId, err)
	}

	return nil
}

func (me *Reconciler) reconcileForActionUserDeactivate(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	userProfile, err := me.connector.GetUserProfileByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("Failed retrieving user profile: %s", err)
	}

	err = me.connector.LogoutAllAccessTokensForUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("Failed logging out all access tokens: %s", err)
	}

	if !matrix.IsUserDeactivatedAccordingToDisplayName(userProfile.DisplayName) {
		newDisplayName := fmt.Sprintf(
			"%s%s",
			matrix.DeactivatedAccountPrefixMarker,
			userProfile.DisplayName,
		)
		err = me.connector.SetUserDisplayName(ctx, userId, newDisplayName)
		if err != nil {
			return fmt.Errorf("Failed setting display name (%s) for %s: %s", newDisplayName, userId, err)
		}
	}

	return nil
}

func (me *Reconciler) reconcileForActionCommunityJoin(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	communityId, err := action.GetStringPayloadDataByKey("communityId")
	if err != nil {
		return err
	}

	err = me.connector.InviteUserToCommunity(ctx, me.reconciliatorUserId, userId, communityId)
	if err != nil {
		return err
	}

	return me.connector.AcceptCommunityInvite(ctx, userId, communityId)
}

func (me *Reconciler) reconcileForActionCommunityLeave(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	communityId, err := action.GetStringPayloadDataByKey("communityId")
	if err != nil {
		return err
	}

	return me.connector.KickUserFromCommunity(ctx, me.reconciliatorUserId, userId, communityId)
}

func (me *Reconciler) reconcileForActionRoomJoin(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	roomId, err := action.GetStringPayloadDataByKey("roomId")
	if err != nil {
		return err
	}

	err = me.connector.InviteUserToRoom(ctx, me.reconciliatorUserId, userId, roomId)
	if err != nil {
		return err
	}

	return me.connector.JoinRoom(ctx, userId, roomId)
}

func (me *Reconciler) reconcileForActionRoomLeave(ctx *connector.AccessTokenContext, action *reconciliation.StateAction) error {
	userId, err := action.GetStringPayloadDataByKey("userId")
	if err != nil {
		return err
	}

	roomId, err := action.GetStringPayloadDataByKey("roomId")
	if err != nil {
		return err
	}

	return me.connector.LeaveRoom(ctx, userId, roomId)
}
