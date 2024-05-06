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

func (*Controller) HandleBroadcast() {
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

func (*Controller) GetWebsocket(c *gin.Context) {
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

		_, err := client.Chat.
			Create().
			SetUsername(chat.Username).
			SetContent(chat.Content).
			Save(ctx)
		if err != nil {
			log.Printf("failed to save chat")
			log.Print(err.Error())
			return
		}

		log.Printf("%s: %s", chat.Username, chat.Content)

		broadcast <- chat
	}
}
