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
//	@Description	And append the access token to the URL as a query parameter, e.g. "ws://localhost:8080/ws?access_token=${accessToken}".
//	@Description
//	@Description	Send and receive messages in JSON format, containing 3 required fields: chatroomId, senderId, and action, and 1 optional field: content.
//	@Description	Only SEND_TEXT action requires the content field.
//	@Description
//	@Description	Action types
//	@Description	JOIN_ROOM: If you receive this action, you should add the sender to the chatroom with a default status (muted and cam off).
//	@Description	LEAVE_ROOM: If you receive this action, you should remove the sender from the chatroom. And you should send this action when you leave the chatroom.
//	@Description	SEND_TEXT: If you receive this action, you should display the content in the chatroom. And you should send this action when you send a message.
//	@Description	MUTE/UNMUTE: If you receive this action, you should mute/unmute the sender. And you should send this action when you mute/unmute yourself.
//	@Description	TURN_ON_CAM/TURN_OFF_CAM: If you receive this action, you should turn on/off the sender's cam. And you should send this action when you turn on/off your cam.
//	@Description	KICKED: If you receive this action, you should know that you are kicked from the chatroom.
//	@Description
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "SEND_TEXT", "content": "Hello, world!"}
//	@Description	Example message: {"chatroomId": 1, "senderId": 1, "action": "MUTE"}
//	@Tags			websocket
//	@Summary		establish a WebSocket connection
//	@Param			access_token	query	string	true	"access token"
//	@Security		BearerAuth
//	@Success		101
//	@Failure		401	"unauthorized"
//	@Response		200	{object}	controller.Message
//	@Router			/ws [get]
func (*Controller) ConnectWebsocket(c *gin.Context) {
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
			hub.clients[client.ID] = client

		case client := <-hub.unregister:
			if _, ok := hub.clients[client.ID]; ok {
				delete(hub.clients, client.ID)
				close(client.send)
			}

		case message := <-hub.broadcast:
			for _, client := range hub.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(hub.clients, client.ID)
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
			room.clients[client.ID] = client

		case client := <-room.unregister:
			delete(room.clients, client.ID)

			if len(room.clients) == 0 {
				delete(hub.rooms, room.id)
				return
			}

		case message := <-room.broadcast:
			for _, client := range room.clients {
				select {
				case client.send <- message:
				default:
					delete(room.clients, client.ID)
				}
			}
		}
	}
}

func broadcast(room *Room, message Message) {
	buf, _ := json.Marshal(message)
	room.broadcast <- buf
}

func joinRoom(roomID, clientID int) []*Client {
	room, ok := hub.rooms[roomID]
	if !ok {
		room = hub.createRoom(roomID)
	}

	client, ok := hub.clients[clientID]
	if ok {
		room.register <- client
		client.room = room
	}

	broadcast(room, Message{
		ChatroomID: roomID,
		SenderID:   clientID,
		Action:     JoinRoomAction,
	})

	clients := make([]*Client, 0, len(room.clients))
	for _, client := range room.clients {
		clients = append(clients, client)
	}

	return clients
}

func kickAllClientsFromRoom(roomID int) {
	room, ok := hub.rooms[roomID]
	if !ok {
		return
	}

	for _, client := range room.clients {
		message, _ := json.Marshal(Message{
			ChatroomID: roomID,
			SenderID:   client.ID,
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
	Content    string `json:"content"`
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
	ID    int             `json:"userId" binding:"required"`
	conn  *websocket.Conn `json:"-"`
	send  chan []byte     `json:"-"`
	room  *Room           `json:"-"`
	Muted bool            `json:"muted" binding:"required"`
	CamOn bool            `json:"camOn" binding:"required"`
}

func newClient(id int, conn *websocket.Conn) *Client {
	client := &Client{
		ID:    id,
		conn:  conn,
		send:  make(chan []byte, 256),
		room:  nil,
		Muted: true,
		CamOn: false,
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
