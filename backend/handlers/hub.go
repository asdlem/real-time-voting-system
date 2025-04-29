// 旧版WebSocket集线器，已被websocket.go替代
// 已弃用，保留做参考
package handlers

/* 旧代码保留作参考
// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients (not used in our case, but required by the gorilla/websocket example).
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// Client represents a connected WebSocket client.
type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// The hub that the client is connected to.
	hub *Hub

	// The poll ID that the client is watching.
	pollID uint
}

// newline is used to split multiple messages in the same writer.
var newline = []byte{'\n'}

// GlobalHub is the global hub for all WebSocket connections.
var GlobalHub = &Hub{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

// PollUpdateMessage represents a message with updated poll results.
type PollUpdateMessage struct {
	PollID  uint                   `json:"poll_id"`
	Results []map[string]interface{} `json:"results"`
}

// run starts the hub and handles client registration/unregistration and message broadcasting.
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("New client registered for poll %d, total clients: %d", client.pollID, len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client unregistered for poll %d, remaining clients: %d", client.pollID, len(h.clients))
			}
		case message := <-h.broadcast:
			var pollUpdate PollUpdateMessage
			if err := json.Unmarshal(message, &pollUpdate); err != nil {
				log.Printf("Error unmarshalling poll update message: %v", err)
				continue
			}

			// Targeted broadcast - only send to clients watching this poll
			for client := range h.clients {
				if client.pollID == pollUpdate.PollID {
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
}

// GetCurrentPollResults fetches the current results for a poll from the database.
func GetCurrentPollResults(pollID uint) ([]map[string]interface{}, error) {
	// This implementation would normally query the database
	// For now, we'll use mock data
	results := []map[string]interface{}{
		{
			"id":         1,
			"text":       "Option A",
			"votes":      15,
			"percentage": 50.0,
		},
		{
			"id":         2,
			"text":       "Option B",
			"votes":      10,
			"percentage": 33.33,
		},
		{
			"id":         3,
			"text":       "Option C",
			"votes":      5,
			"percentage": 16.67,
		},
	}
	return results, nil
}

// BroadcastPollUpdate broadcasts poll update to all connected clients watching this poll.
func BroadcastPollUpdate(pollID uint, results []map[string]interface{}) {
	// Construct the update message
	updateMsg := PollUpdateMessage{
		PollID:  pollID,
		Results: results,
	}

	// Convert to JSON
	jsonMsg, err := json.Marshal(updateMsg)
	if err != nil {
		log.Printf("Error marshalling poll update: %v", err)
		return
	}

	// Send to the hub for broadcasting
	GlobalHub.broadcast <- jsonMsg
}
*/

// 初始化函数，用于启动hub
func init() {
	// 注释掉GlobalHub.run()，因为已经在websocket.go中实现
	// go GlobalHub.run()
}
