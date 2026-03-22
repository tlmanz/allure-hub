package usecase

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/domain"
)

// EventBus is an in-memory fan-out broadcaster for upload session events.
// Each SSE connection subscribes to receive session_updated messages.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]chan []byte
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: make(map[string]chan []byte)}
}

// Subscribe registers a new SSE listener and returns its ID and read channel.
func (b *EventBus) Subscribe() (id string, ch <-chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id = uuid.New().String()
	c := make(chan []byte, 64)
	b.subscribers[id] = c
	return id, c
}

// Unsubscribe removes a listener and closes its channel.
func (b *EventBus) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c, ok := b.subscribers[id]; ok {
		close(c)
		delete(b.subscribers, id)
	}
}

// Shutdown closes every subscriber channel so all SSE handlers return promptly,
// allowing the HTTP server to drain cleanly during graceful shutdown.
func (b *EventBus) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, c := range b.subscribers {
		close(c)
		delete(b.subscribers, id)
	}
}

// Publish marshals the session as an SSE event and fans it out to all listeners.
// Slow listeners are skipped (non-blocking send) to avoid head-of-line blocking.
func (b *EventBus) Publish(session *domain.UploadSession) {
	data, err := json.Marshal(session)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: session_updated\ndata: %s\n\n", data))

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, c := range b.subscribers {
		select {
		case c <- msg:
		default: // drop for slow consumers
		}
	}
}
