package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { 
		return true // Allow all origins in development
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

var hub = Hub{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

type ChatMessage struct {
	Type      string    `json:"type"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	File      *FileInfo `json:"file,omitempty"`
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client connected. Total clients: %d", len(h.clients))
			
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected. Total clients: %d", len(h.clients))
			}
			
		case message := <-h.broadcast:
			var msg ChatMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Error unmarshaling chat message: %v", err)
				continue
			}
			
			// Save to database
			if err := insertChatMessage(msg); err != nil {
				log.Printf("Error inserting chat message: %v", err)
			}
			
			// Broadcast to all clients
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		
		var chatMsg ChatMessage
		if err := json.Unmarshal(message, &chatMsg); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}
		
		// Validate message
		if chatMsg.Message == "" && chatMsg.File == nil {
			continue
		}
		
		// Set timestamp if not provided
		if chatMsg.Timestamp.IsZero() {
			chatMsg.Timestamp = time.Now()
		}
		
		// Set default type
		if chatMsg.Type == "" {
			chatMsg.Type = "message"
		}
		
		messageData, err := json.Marshal(chatMsg)
		if err != nil {
			log.Printf("Error marshaling chat message: %v", err)
			continue
		}
		
		c.hub.broadcast <- messageData
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			
			if _, err := w.Write(message); err != nil {
				return
			}
			
			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write(<-c.send); err != nil {
					return
				}
			}
			
			if err := w.Close(); err != nil {
				return
			}
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	
	client := &Client{
		hub:  &hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	
	client.hub.register <- client
	
	go client.writePump()
	go client.readPump()
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method == "GET" {
		messages := getAllChatMessages()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(messages); err != nil {
			log.Printf("Error encoding chat messages: %v", err)
			http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		}
		return
	}
	
	if r.Method == "POST" {
		var message ChatMessage
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, "Invalid message", http.StatusBadRequest)
			return
		}
		
		// Validate
		if message.Message == "" {
			http.Error(w, "Message cannot be empty", http.StatusBadRequest)
			return
		}
		
		if message.Username == "" {
			message.Username = "Anonymous"
		}
		
		message.Timestamp = time.Now()
		message.Type = "message"
		
		messageData, err := json.Marshal(message)
		if err != nil {
			http.Error(w, "Error marshaling message", http.StatusInternalServerError)
			return
		}
		
		hub.broadcast <- messageData
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}