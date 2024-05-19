package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Message struct {
	ChatroomID int    `json:"chatroomId"`
	SenderID   int    `json:"senderId"`
	Content    string `json:"content"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

func (*Controller) HandleBroadcast() {
	for {
		msg := <-broadcast

		for client := range clients {
			if err := client.WriteJSON(msg); err != nil {
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
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("conn.ReadMessage: %v", err)
			return
		}

		_, err := client.Chat.
			Create().
			SetChatroomID(msg.ChatroomID).
			SetSenderID(msg.SenderID).
			SetContent(msg.Content).
			Save(ctx)
		if err != nil {
			log.Printf("failed to save chat")
			log.Print(err.Error())
			return
		}

		log.Printf("%d: %s", msg.SenderID, msg.Content)

		broadcast <- msg
	}
}
