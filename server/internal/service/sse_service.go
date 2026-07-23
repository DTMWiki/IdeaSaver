package service

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID     string
	UserID uuid.UUID
	Events chan SSEEvent
}

const maxSSEConnectionsPerUser = 5

// SSEService manages SSE connections and event broadcasting.
type SSEService struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
}

func NewSSEService() *SSEService {
	return &SSEService{
		clients: make(map[string]*SSEClient),
	}
}

// Register adds a new SSE client. Returns false if the user already has too many connections.
func (s *SSEService) Register(client *SSEClient) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client == nil {
		return false
	}
	count := 0
	for _, c := range s.clients {
		if c.UserID == client.UserID {
			count++
		}
	}
	if count >= maxSSEConnectionsPerUser {
		return false
	}
	s.clients[client.ID] = client
	return true
}

// Unregister removes an SSE client.
func (s *SSEService) Unregister(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client, ok := s.clients[clientID]; ok {
		close(client.Events)
		delete(s.clients, clientID)
	}
}

// SendToUser sends an event to all connected clients of a specific user.
func (s *SSEService) SendToUser(userID uuid.UUID, event SSEEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.UserID == userID {
			select {
			case client.Events <- event:
			default:
				// Channel full, skip
			}
		}
	}
}

// Broadcast sends an event to all connected clients.
func (s *SSEService) Broadcast(event SSEEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		select {
		case client.Events <- event:
		default:
		}
	}
}

// FormatSSE formats an SSEEvent for HTTP streaming.
func FormatSSE(event SSEEvent) string {
	data, _ := json.Marshal(event.Data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}
