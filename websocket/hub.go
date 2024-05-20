package ws

var hub = newHub()

type Hub struct {
	rooms      map[int]*Room
	clients    map[int]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func newHub() *Hub {
	hub := &Hub{
		rooms:      make(map[int]*Room),
		clients:    make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}

	go hub.run()

	return hub
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
