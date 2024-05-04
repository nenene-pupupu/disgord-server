package controller

import (
	"log"
	"net/http"

	"disgord/ent"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan ent.Chat)

func handleBroadcast() {
	for {
		chat := <-broadcast

		for client := range clients {
			if err := client.WriteJSON(chat); err != nil {
				log.Printf("client.WriteMessage: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func (controller Controller) InitChat() {
	group := controller.Router.Group("/chat")

	group.GET("", func(c *gin.Context) {
		chats, err := controller.Client.Chat.
			Query().
			All(controller.Ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, chats)
	})

	group.GET("/ws", func(c *gin.Context) {
		if len(clients) >= 6 {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "too many participants in this room",
			})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("upgrader.Upgrade: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("Client connected: %v", conn.RemoteAddr())
		clients[conn] = true
		defer delete(clients, conn)

		for {
			var chat ent.Chat
			if err := conn.ReadJSON(&chat); err != nil {
				log.Printf("conn.ReadMessage: %v", err)
				return
			}

			_, err := controller.Client.Chat.
				Create().
				SetUsername(chat.Username).
				SetContent(chat.Content).
				Save(controller.Ctx)
			if err != nil {
				log.Printf("failed to save chat")
				return
			}

			log.Printf("%s: %s", chat.Username, chat.Content)

			broadcast <- chat
		}
	})

	go handleBroadcast()

	// group.GET("/:id/ws/text", func(ctx *gin.Context) {})
	// group.GET("/:id/ws/voice", func(ctx *gin.Context) {})
	// group.GET("/:id/ws/video", func(ctx *gin.Context) {})

}
