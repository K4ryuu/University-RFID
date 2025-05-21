package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
	UserID  uint        `json:"user_id,omitempty"`
	Admin   bool        `json:"admin,omitempty"`
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    uint
	isAdmin   bool
	mu        sync.Mutex
	isClosing bool
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Új WebSocket kliens csatlakozott (UserID: %d, Admin: %v)", client.userID, client.isAdmin)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("WebSocket kliens lecsatlakozott (UserID: %d, Admin: %v)", client.userID, client.isAdmin)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToAll(messageType string, content interface{}) {
	message := Message{
		Type:    messageType,
		Content: content,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Hiba a WebSocket üzenet készítése közben: %v", err)
		return
	}

	h.broadcast <- data
}

func (h *Hub) BroadcastToAdmins(messageType string, content interface{}) {
	message := Message{
		Type:    messageType,
		Content: content,
		Admin:   true,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Hiba a WebSocket üzenet készítése közben: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.isAdmin {
			select {
			case client.send <- data:
			default:
				continue
			}
		}
	}
}

func (h *Hub) BroadcastToUser(userID uint, messageType string, content interface{}) {
	message := Message{
		Type:    messageType,
		Content: content,
		UserID:  userID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Hiba a WebSocket üzenet készítése közben: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- data:
			default:
				continue
			}
		}
	}
}

func (h *Hub) BroadcastToAuthenticated(messageType string, content interface{}) {
	message := Message{
		Type:    messageType,
		Content: content,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Hiba a WebSocket üzenet készítése közben: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.userID > 0 {
			select {
			case client.send <- data:
			default:
				continue
			}
		}
	}
}

func (h *Hub) SendAccessEvent(event map[string]interface{}) {
	h.BroadcastToAdmins("access_event", event)

	if userID, ok := event["user_id"].(uint); ok {
		h.BroadcastToUser(userID, "access_event", event)
	}
}

func (client *Client) HandleClientConnection() {
	client.hub.register <- client

	defer func() {
		client.mu.Lock()
		client.isClosing = true
		client.mu.Unlock()
		client.hub.unregister <- client
		client.conn.Close()
	}()

	go client.writePump()
	client.readPump()
}

func (client *Client) readPump() {
	defer func() {
		client.mu.Lock()
		if !client.isClosing {
			client.hub.unregister <- client
			client.conn.Close()
		}
		client.mu.Unlock()
	}()

	client.conn.SetReadLimit(1024)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket hiba: %v", err)
			}
			break
		}
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.mu.Lock()
		if !client.isClosing {
			client.conn.Close()
		}
		client.mu.Unlock()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}