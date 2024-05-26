package ws

const (
	JoinRoomAction  = "JOIN_ROOM"
	LeaveRoomAction = "LEAVE_ROOM"

	SendTextAction = "SEND_TEXT"

	MuteAction   = "MUTE"
	UnmuteAction = "UNMUTE"

	TurnOnCamAction  = "TURN_ON_CAM"
	TurnOffCamAction = "TURN_OFF_CAM"
)

type Message struct {
	ChatroomID int    `json:"chatroomId"`
	SenderID   int    `json:"senderId"`
	Action     string `json:"action"`
	Content    string `json:"content,omitempty"`
}
