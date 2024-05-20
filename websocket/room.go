package ws

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
