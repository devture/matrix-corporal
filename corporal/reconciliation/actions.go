package reconciliation

// All reconciliation actions must have a corresponding
// reconciliation handler function in `reconciler.go`.
const (
	ActionUserCreate         = "user.create"
	ActionUserSetDisplayName = "user.set_display_name"
	ActionUserSetAvatar      = "user.set_avatar"
	ActionUserActivate       = "user.activate"
	ActionUserDeactivate     = "user.deactivate"

	ActionCommunityJoin  = "community.join"
	ActionCommunityLeave = "community.leave"

	ActionRoomJoin  = "room.join"
	ActionRoomLeave = "room.leave"
)
