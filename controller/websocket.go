package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
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
//	@Description	Send and receive messages in JSON format.
//	@Description
//	@Description	When you send a message to the server:
//	@Description	You can use action types: LIST_USERS, LEAVE_ROOM, SEND_TEXT, MUTE, UNMUTE, TURN_ON_CAM, TURN_OFF_CAM.
//	@Description	Especially, SEND_TEXT should contain the content field.
//	@Description
//	@Description	When you receive a message from the server:
//	@Description	If you send LIST_USERS, you will receive LIST_USERS with a list of users in the chatroom.
//	@Description	If any user sends SEND_TEXT, you will receive the same message.
//	@Description	If any user sends other action messages, you will receive LIST_USERS with a list of users in the chatroom.
//	@Description	If you receive KICKED, you should know that you are kicked from the chatroom.
//	@Description	If you receive INVALID, you should know that the message you sent is invalid.
//	@Description
//	@Description	To connect WebRTC, if you receive OFFER with offer content, you should send ANSWER with answer content.
//	@Description	Then, if you send CANDIDATE with candidate content, you will receive CANDIDATE with candidate content.
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
}

var hub *Hub

func init() {
	hub = &Hub{
		rooms:      make(map[int]*Room),
		clients:    make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go hub.run()
}

func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.clients[client.ID] = client

		case client := <-hub.unregister:
			if client.room != nil {
				client.room.unregister <- client
			}

			if _, ok := hub.clients[client.ID]; ok {
				delete(hub.clients, client.ID)
				close(client.send)
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
	id          int
	clients     map[int]*Client
	register    chan *Client
	unregister  chan *Client
	broadcast   chan *Message
	listLock    sync.RWMutex
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
}

func newRoom(id int) *Room {
	room := &Room{
		id:          id,
		clients:     make(map[int]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
		listLock:    sync.RWMutex{},
		trackLocals: map[string]*webrtc.TrackLocalStaticRTP{},
	}

	go room.run()

	go func() {
		for range time.NewTicker(time.Second * 3).C {
			room.dispatchKeyFrame()
		}
	}()

	return room
}

func (room *Room) run() {
	for {
		select {
		case client := <-room.register:
			room.clients[client.ID] = client
			client.room = room
			client.connectToPeers(room)

			room.broadcast <- room.listClients()

		case client := <-room.unregister:
			client.pc.Close()
			client.pc = nil

			delete(room.clients, client.ID)
			client.room = nil

			if len(room.clients) == 0 {
				delete(hub.rooms, room.id)
				return
			}

			room.broadcast <- room.listClients()

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

func joinRoom(roomID, clientID int, muted, camOn bool) {
	room, ok := hub.rooms[roomID]
	if !ok {
		room = hub.createRoom(roomID)
	}

	client, ok := hub.clients[clientID]
	if ok {
		client.Muted = muted
		client.CamOn = camOn
		room.register <- client
	}
}

func (room *Room) listClients() *Message {
	clients := make([]*Client, 0, len(room.clients))
	for _, client := range room.clients {
		clients = append(clients, client)
	}

	b, _ := json.Marshal(clients)

	return &Message{
		ChatroomID: room.id,
		Action:     ListUsersAction,
		Content:    string(b),
	}
}

func kickAllClientsFromRoom(roomID int) {
	room, ok := hub.rooms[roomID]
	if !ok {
		return
	}

	message := &Message{
		ChatroomID: roomID,
		Action:     KickedAction,
	}

	for _, client := range room.clients {
		client.send <- message
		room.unregister <- client
	}
}

const (
	ListUsersAction = "LIST_USERS"

	JoinRoomAction  = "JOIN_ROOM"
	LeaveRoomAction = "LEAVE_ROOM"

	SendTextAction = "SEND_TEXT"

	MuteAction   = "MUTE"
	UnmuteAction = "UNMUTE"

	TurnOnCamAction  = "TURN_ON_CAM"
	TurnOffCamAction = "TURN_OFF_CAM"

	KickedAction = "KICKED"

	OfferAction     = "OFFER"
	AnswerAction    = "ANSWER"
	CandidateAction = "CANDIDATE"

	InvalidAction = "INVALID"
)

type Message struct {
	ChatroomID int       `json:"chatroomId,omitempty"`
	SenderID   int       `json:"senderId,omitempty"`
	Action     string    `json:"action" binding:"required"`
	Content    string    `json:"content,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 65536
)

type Client struct {
	ID    int             `json:"userId" binding:"required"`
	conn  *websocket.Conn `json:"-"`
	send  chan *Message   `json:"-"`
	room  *Room           `json:"-"`
	Muted bool            `json:"muted" binding:"required"`
	CamOn bool            `json:"camOn" binding:"required"`
	pc    *webrtc.PeerConnection
}

func newClient(id int, conn *websocket.Conn) *Client {
	client := &Client{
		ID:    id,
		conn:  conn,
		send:  make(chan *Message, 256),
		Muted: false,
		CamOn: false,
	}

	hub.register <- client

	return client
}

func saveChat(message *Message) {
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

	defer func() { hub.unregister <- client }()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var message *Message
		err := client.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if client.room == nil {
			log.Println("the client is not in a room, message ignored")
			continue
		}
		room := client.room

		message.ChatroomID = room.id
		message.SenderID = client.ID
		message.CreatedAt = time.Now()

		pretty, _ := json.MarshalIndent(message, "", "  ")
		log.Println(string(pretty))

		switch message.Action {
		case ListUsersAction:
			client.send <- room.listClients()

		case LeaveRoomAction:
			room.unregister <- client

		case SendTextAction:
			saveChat(message)
			room.broadcast <- message

		case MuteAction:
			client.Muted = true
			room.broadcast <- room.listClients()

		case UnmuteAction:
			client.Muted = false
			room.broadcast <- room.listClients()

		case TurnOnCamAction:
			client.CamOn = true
			room.broadcast <- room.listClients()

		case TurnOffCamAction:
			client.CamOn = false
			room.broadcast <- room.listClients()

		case AnswerAction:
			answer := webrtc.SessionDescription{}
			json.Unmarshal([]byte(message.Content), &answer)
			client.pc.SetRemoteDescription(answer)

		case CandidateAction:
			candidate := webrtc.ICECandidateInit{}
			json.Unmarshal([]byte(message.Content), &candidate)
			client.pc.AddICECandidate(candidate)

		default:
			buf, _ := json.MarshalIndent(message, "", "  ")

			client.send <- &Message{
				Action:  InvalidAction,
				Content: string(buf),
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

			p, err := json.Marshal(message)
			if err != nil {
				return
			}
			w.Write(p)

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

func (client *Client) connectToPeers(room *Room) {
	// Create new PeerConnection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Print(err)
		return
	}

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := pc.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	client.pc = pc

	// Trickle ICE. Emit server candidate to client
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}

		client.send <- &Message{
			Action:  CandidateAction,
			Content: string(candidateString),
		}
	})

	// If PeerConnection is closed remove it from global list
	pc.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := pc.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			room.signalPeerConnections()
		default:
		}
	})

	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		trackLocal := room.addTrack(t)
		defer room.removeTrack(trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	// Signal for the new PeerConnection
	room.signalPeerConnections()
}
