package ws

import (
	"encoding/json"

	gorillaws "github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[*gorillaws.Conn]bool
	broadcast  chan Event
	register   chan *gorillaws.Conn
	unregister chan *gorillaws.Conn
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*gorillaws.Conn]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *gorillaws.Conn),
		unregister: make(chan *gorillaws.Conn),
	}
}

func (h *Hub) Register(c *gorillaws.Conn) {
	h.register <- c
}

func (h *Hub) Unregister(c *gorillaws.Conn) {
	h.unregister <- c
}

func (h *Hub) Broadcast(e Event) {
	h.broadcast <- e
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			delete(h.clients, c)
		case e := <-h.broadcast:
			msg, _ := json.Marshal(e)
			for c := range h.clients {
				if err := c.WriteMessage(gorillaws.TextMessage, msg); err != nil {
					delete(h.clients, c)
					c.Close()
				}
			}
		}
	}
}
