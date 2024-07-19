package reconciliation

// All reconciliation actions must have a corresponding
// reconciliation handler function in `reconciler.go`.
const (
	ActionUserCreate         = "user.create"
	ActionUserSetDisplayName = "user.set_display_name"
	ActionUserSetAvatar      = "user.set_avatar"
	ActionUserActivate       = "user.activate"
	ActionUserDeactivate     = "user.deactivate"

	ActionRoomJoin                = "room.join"
	ActionRoomLeave               = "room.leave"
	ActionRoomUserSetPowerLevel   = "room.user_set_power_level"
	ActionRoomUsersSetPowerLevels = "room.users_set_power_levels"
)
