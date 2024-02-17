package ws

import "megachat/internal/app/ds"

func NewHub() *Hub {
	return &Hub{
		ds.Hub{
			Broadcast:  make(chan []byte),
			Register:   make(chan *ds.Client),
			Unregister: make(chan *ds.Client),
			Clients:    make(map[*ds.Client]bool),
		},
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}
