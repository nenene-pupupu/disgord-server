package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var newline = []byte{'\n'}

type Client struct {
	id   int
	conn *websocket.Conn
	send chan []byte
	room *Room
}

func NewClient(id int, conn *websocket.Conn) *Client {
	client := &Client{
		id:   id,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	return client
}

// ReadPump pumps messages from the websocket connection to the hub.
//
// The application runs ReadPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (client *Client) ReadPump() {
	defer client.conn.Close()

	defer func() {
		if client.room != nil {
			client.room.unregister <- client
		}
		hub.unregister <- client
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var message Message
		err := client.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		switch message.Action {
		case JoinRoomAction:
			room, ok := hub.rooms[message.ChatroomID]
			if !ok {
				room = hub.createRoom(message.ChatroomID)
			}

			room.register <- client
			client.room = room

		case LeaveRoomAction:
			if client.room != nil {
				client.room.unregister <- client
				client.room = nil
			}

		default:
			if client.room != nil {
				buf, _ := json.Marshal(message)
				client.room.broadcast <- buf
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
//
// A goroutine running WritePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (client *Client) WritePump() {
	defer client.conn.Close()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
