package connector

type CurrentState struct {
	Users []CurrentUserState `json:"users"`
}

func (me *CurrentState) GetUserStateByUserId(userId string) *CurrentUserState {
	for _, userState := range me.Users {
		if userState.Id == userId {
			return &userState
		}
	}
	return nil
}

type CurrentUserRoomState struct {
	RoomId     string `json:"roomId"`
	PowerLevel int    `json:"powerLevel"`
}

type CurrentUserState struct {
	Id                  string                 `json:"id"`
	Active              bool                   `json:"active"`
	DisplayName         string                 `json:"displayName"`
	AvatarMxcUri        string                 `json:"avatarMxcUri"`
	AvatarSourceUriHash string                 `json:"avatarSourceUriHash"`
	JoinedRooms         []CurrentUserRoomState `json:"joinedRooms"`
}
