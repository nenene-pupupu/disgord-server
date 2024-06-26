package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"disgord/ent"

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
//	@Description	If you receive ROOM_LIST_UPDATED, you should update chatroom list with the API.
//	@Description	If you receive INVALID, you should know that the message you sent is invalid.
//	@Description
//	@Description	To connect WebRTC, if you receive OFFER with offer content, you should send ANSWER with answer content.
//	@Description	Then, if you send CANDIDATE with candidate content, you will receive CANDIDATE with candidate content.
//	@Tags			websocket
//	@Summary		establish a WebSocket connection
//	@Param			access_token	query	string	true	"access token"
//	@Security		BearerAuth
//	@Success		101
//	@Failure		401
//	@Failure		404		"cannot find user"
//	@Response		1000	{object}	controller.Message				"SEND_TEXT message format"
//	@Response		1001	{object}	controller.ListClients.Response	"LIST_USERS content format"
//	@Router			/ws [get]
func (*Controller) ConnectWebsocket(c *gin.Context) {
	userID := getCurrentUserID(c)

	user, err := client.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, user)

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

	go func() {
		for range time.NewTicker(time.Second * 3).C {
			for _, room := range hub.rooms {
				go room.dispatchKeyFrame()
			}
		}
	}()
}

func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			if _, ok := hub.clients[client.ID]; ok {
				close(client.send)
				continue
			}

			hub.clients[client.ID] = client

		case client := <-hub.unregister:
			if client.room != nil {
				client.room.unregister <- client
			}

			if client == hub.clients[client.ID] {
				delete(hub.clients, client.ID)
				close(client.send)
			}
		}
	}
}

func broadcastToAll(message *Message) {
	for _, client := range hub.clients {
		client.send <- message
	}
}

func disconnect(clientID int) {
	client, ok := hub.clients[clientID]
	if ok {
		hub.unregister <- client
	}
}

type Room struct {
	id          int
	clients     map[int]*Client
	register    chan *Client
	unregister  chan *Client
	broadcast   chan *Message
	listLock    sync.RWMutex
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	sidTable    map[int]string
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
		sidTable:    map[int]string{},
	}

	go room.run()

	return room
}

func (room *Room) run() {
	for {
		select {
		case client := <-room.register:
			room.clients[client.ID] = client
			client.room = room
			client.connectToPeers(room)

			room.broadcast <- room.ListClients()

		case client := <-room.unregister:
			delete(room.clients, client.ID)
			client.room = nil

			client.pc.Close()
			client.pc = nil

			if len(room.clients) == 0 {
				delete(hub.rooms, room.id)
				return
			}

			room.broadcast <- room.ListClients()

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
		room = newRoom(roomID)
		hub.rooms[roomID] = room
	}

	client, ok := hub.clients[clientID]
	if ok {
		client.Muted = muted
		client.CamOn = camOn
		room.register <- client
	}
}

func kickAllClientsFromRoom(roomID int) {
	room, ok := hub.rooms[roomID]
	if !ok {
		return
	}

	message := &Message{Action: KickedAction}
	for _, client := range room.clients {
		client.send <- message
		room.unregister <- client
	}
}

func (room *Room) ListClients() *Message {
	keys := make([]int, 0, len(room.clients))
	for k := range room.clients {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	type Response struct {
		*Client
		StreamID string `json:"streamId,omitempty"`
	}

	response := make([]*Response, 0, len(keys))
	for _, k := range keys {
		response = append(response, &Response{room.clients[k], room.sidTable[k]})
	}

	b, _ := json.Marshal(response)

	return &Message{
		Action:  ListUsersAction,
		Content: string(b),
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

	RoomListUpdatedAction = "ROOM_LIST_UPDATED"

	OfferAction     = "OFFER"
	AnswerAction    = "ANSWER"
	CandidateAction = "CANDIDATE"

	InvalidAction = "INVALID"
)

type Message struct {
	Action    string     `json:"action" binding:"required"`
	Content   string     `json:"content,omitempty"`
	Name      string     `json:"displayName,omitempty"`
	Color     uint8      `json:"profileColorIndex,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
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
	ID    int                    `json:"userId" binding:"required"`
	conn  *websocket.Conn        `json:"-"`
	send  chan *Message          `json:"-"`
	room  *Room                  `json:"-"`
	pc    *webrtc.PeerConnection `json:"-"`
	Name  string                 `json:"displayName" binding:"required"`
	Color uint8                  `json:"profileColorIndex" binding:"required"`
	Muted bool                   `json:"muted" binding:"required"`
	CamOn bool                   `json:"camOn" binding:"required"`
}

func newClient(conn *websocket.Conn, user *ent.User) *Client {
	client := &Client{
		ID:    user.ID,
		conn:  conn,
		send:  make(chan *Message, 256),
		Name:  user.DisplayName,
		Color: user.ProfileColorIndex,
		Muted: false,
		CamOn: false,
	}

	hub.register <- client

	return client
}

func saveChat(chatroomID, senderID int, content string) {
	client.Chat.
		Create().
		SetChatroomID(chatroomID).
		SetSenderID(senderID).
		SetContent(content).
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
		message := &Message{}
		err := client.conn.ReadJSON(message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		message.Name = client.Name
		message.Color = client.Color
		message.CreatedAt = &time.Time{}
		*message.CreatedAt = time.Now()

		pretty, _ := json.MarshalIndent(message, "", "  ")
		log.Println(string(pretty))

		if client.room == nil {
			log.Println("the client is not in a room, message ignored")
			continue
		}
		room := client.room

		switch message.Action {
		case ListUsersAction:
			client.send <- room.ListClients()

		case LeaveRoomAction:
			room.unregister <- client

		case SendTextAction:
			saveChat(room.id, client.ID, message.Content)
			room.broadcast <- message

		case MuteAction:
			client.Muted = true
			room.broadcast <- room.ListClients()

		case UnmuteAction:
			client.Muted = false
			room.broadcast <- room.ListClients()

		case TurnOnCamAction:
			client.CamOn = true
			room.broadcast <- room.ListClients()

		case TurnOffCamAction:
			client.CamOn = false
			room.broadcast <- room.ListClients()

		case AnswerAction:
			answer := webrtc.SessionDescription{}
			json.Unmarshal([]byte(message.Content), &answer)
			client.pc.SetRemoteDescription(answer)

		case CandidateAction:
			candidate := webrtc.ICECandidateInit{}
			json.Unmarshal([]byte(message.Content), &candidate)
			client.pc.AddICECandidate(candidate)

		default:
			client.send <- &Message{
				Action:  InvalidAction,
				Content: string(pretty),
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
		trackLocal := room.addTrack(t, client.ID)
		room.broadcast <- room.ListClients()
		defer room.removeTrack(trackLocal, client.ID)

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
