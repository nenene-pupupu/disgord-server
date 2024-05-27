package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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
//	@Description	Actions: [JOIN_ROOM, LEAVE_ROOM, SEND_TEXT, MUTE, UNMUTE, TURN_ON_CAM, TURN_OFF_CAM, KICKED]
//	@Description	Only SEND_TEXT action requires the content field.
//	@Description	JOIN_ROOM and LEAVE_ROOM actions let the server know which chatroom the client is in.
//	@Description	SEND_TEXT action and the other status-related actions will be broadcasted to all clients in the same chatroom.
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "SEND_TEXT", "content": "Hello, world!"}
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "MUTE"}
//	@Tags			websocket
//	@Summary		establish a WebSocket connection
//	@Param			Authorization	header	string	true	"Bearer AccessToken"
//	@Security		BearerAuth
//	@Success		101
//	@Failure		400	"invalid scheme"
//	@Failure		401	"unauthorized"
//	@Response		200	{object}	controller.Message
//	@Router			/ws [get]
func (*Controller) ConnectWebsocket(c *gin.Context) {
	if c.Request.URL.Scheme != "ws" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid scheme",
		})
		return
	}

	userID := getCurrentUserID(c)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(userID, conn)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

type Hub struct {
	rooms      map[int]*Room
	clients    map[int]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

var hub *Hub

func init() {
	hub = &Hub{
		rooms:      make(map[int]*Room),
		clients:    make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}

	go hub.run()
}

func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.clients[client.id] = client

		case client := <-hub.unregister:
			if _, ok := hub.clients[client.id]; ok {
				delete(hub.clients, client.id)
				close(client.send)
			}

		case message := <-hub.broadcast:
			for _, client := range hub.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(hub.clients, client.id)
				}
			}
		}
	}
}

func (hub *Hub) createRoom(id int) (room *Room) {
	room = newRoom(id)
	hub.rooms[id] = room
	return
}

type Room struct {
	id         int
	clients    map[int]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func newRoom(id int) *Room {
	room := &Room{
		id:         id,
		clients:    make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}

	go room.run()

	return room
}

func (room *Room) run() {
	for {
		select {
		case client := <-room.register:
			room.clients[client.id] = client

		case client := <-room.unregister:
			delete(room.clients, client.id)

			if len(room.clients) == 0 {
				return
			}

		case message := <-room.broadcast:
			for _, client := range room.clients {
				select {
				case client.send <- message:
				default:
					delete(room.clients, client.id)
				}
			}
		}
	}
}

func kickAllClientsFromRoom(roomID int) {
	room, ok := hub.rooms[roomID]
	if !ok {
		return
	}

	for _, client := range room.clients {
		message, _ := json.Marshal(Message{
			ChatroomID: roomID,
			SenderID:   client.id,
			Action:     KickedAction,
		})
		client.send <- message

		room.unregister <- client
		client.room = nil
	}
}

const (
	JoinRoomAction  = "JOIN_ROOM"
	LeaveRoomAction = "LEAVE_ROOM"

	SendTextAction = "SEND_TEXT"

	MuteAction   = "MUTE"
	UnmuteAction = "UNMUTE"

	TurnOnCamAction  = "TURN_ON_CAM"
	TurnOffCamAction = "TURN_OFF_CAM"

	KickedAction = "KICKED"
)

type Message struct {
	ChatroomID int    `json:"chatroomId" binding:"required"`
	SenderID   int    `json:"senderId" binding:"required"`
	Action     string `json:"action" binding:"required"`
	Content    string `json:"content,omitempty"`
}

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

func newClient(id int, conn *websocket.Conn) *Client {
	client := &Client{
		id:   id,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	return client
}

func saveChat(message Message) {
	client.Chat.
		Create().
		SetChatroomID(message.ChatroomID).
		SetSenderID(message.SenderID).
		SetContent(message.Content).
		Save(ctx)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (client *Client) readPump() {
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

		case SendTextAction:
			saveChat(message)
			fallthrough

		default:
			if client.room != nil {
				buf, _ := json.Marshal(message)
				client.room.broadcast <- buf
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (client *Client) writePump() {
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
