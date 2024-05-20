package ws

const (
	JoinRoomAction  = "JOIN_ROOM"
	LeaveRoomAction = "LEAVE_ROOM"

	SendTextAction  = "SEND_TEXT"
	SendVoiceAction = "SEND_VOICE"
	SendVideoAction = "SEND_VIDEO"

	MuteAction   = "MUTE"
	UnmuteAction = "UNMUTE"

	DeafenAction   = "DEAFEN"
	UndeafenAction = "UNDEAFEN"

	TurnOnCamAction  = "TURN_ON_CAM"
	TurnOffCamAction = "TURN_OFF_CAM"
)

type Message struct {
	ChatroomID int    `json:"chatroomId"`
	SenderID   int    `json:"senderId"`
	Action     string `json:"action"`
	Data       string `json:"data,omitempty"`
}
