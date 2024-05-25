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
// @Tags	websocket
// @Router	/ws [get]
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
			"message": "Unauthorized",
		})
		return
	}

	user, err := client.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "user not found",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := ws.NewClient(userID, user.DisplayName, conn)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
}
