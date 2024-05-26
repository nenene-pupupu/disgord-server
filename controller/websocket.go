package controller

import (
	"log"
	"net/http"

	"disgord/jwt"
	ws "disgord/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ConnectWebsocket godoc
//
//	@Description	Use the ws:// scheme instead of the http:// scheme to establish a WebSocket connection.
//	@Description	Send and receive messages in JSON format, containing 3 required fields: chatroomId, senderId, and action, and 1 optional field: content.
//	@Description	Actions: [JOIN_ROOM, LEAVE_ROOM, SEND_TEXT, MUTE, UNMUTE, TURN_ON_CAM, TURN_OFF_CAM]
//	@Description	Only SEND_TEXT action requires the content field.
//	@Description	JOIN_ROOM and LEAVE_ROOM actions let the server know which chatroom the client is in.
//	@Description	SEND_TEXT action and the other status-related actions will be broadcasted to all clients in the same chatroom.
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "SEND_TEXT", "content": "Hello, world!"}
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "MUTE"}
//	@Tags			websocket
//	@Param			Authorization	header	string	true	"Bearer AccessToken"
//	@Security		BearerAuth
//	@Success		101
//	@Failure		400	"invalid scheme"
//	@Failure		401	"unauthorized"
//	@Response		200	{object}	ws.Message
//	@Router			/ws [get]
func (*Controller) ConnectWebsocket(c *gin.Context) {
	if c.Request.URL.Scheme != "ws" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid scheme",
		})
		return
	}

	userID, ok := jwt.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := ws.NewClient(userID, conn)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
}
