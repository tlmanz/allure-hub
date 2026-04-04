package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/tlmanz/allure-hub/internal/domain"
	notify "github.com/tlmanz/go-notify"
)

// NotificationSender is the minimal notifier interface EventBus needs.
type NotificationSender interface {
	Send(ctx context.Context, notif notify.Notification) error
}

// EventBus is an in-memory fan-out broadcaster for upload session events.
// Each SSE connection subscribes to receive session_updated messages.
type EventBus struct {
	mu                  sync.RWMutex
	subscribers         map[string]chan []byte
	notifier            NotificationSender
	lastTerminalByBuild map[string]domain.UploadPhase
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers:         make(map[string]chan []byte),
		lastTerminalByBuild: make(map[string]domain.UploadPhase),
	}
}

// SetNotifier configures the optional notifier sink used for app notifications.
func (b *EventBus) SetNotifier(n NotificationSender) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.notifier = n
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
	clear(b.lastTerminalByBuild)
}

// Publish marshals the session as an SSE event and fans it out to all listeners.
// Slow listeners are skipped (non-blocking send) to avoid head-of-line blocking.
func (b *EventBus) Publish(session *domain.UploadSession) {
	if session == nil {
		return
	}

	data, err := json.Marshal(session)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: session_updated\ndata: %s\n\n", data))

	b.mu.RLock()
	for _, c := range b.subscribers {
		select {
		case c <- msg:
		default: // drop for slow consumers
		}
	}
	b.mu.RUnlock()

	notif, sender, ok := b.notificationForSession(session)
	if !ok || sender == nil {
		return
	}
	_ = sender.Send(context.Background(), notif)
}

func (b *EventBus) notificationForSession(session *domain.UploadSession) (notify.Notification, NotificationSender, bool) {
	phase := session.Phase
	if phase != domain.PhaseDone && phase != domain.PhaseFailed {
		return notify.Notification{}, nil, false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if session.BuildID != "" {
		// Keep dedupe state bounded for long-running processes.
		if len(b.lastTerminalByBuild) > 5000 {
			clear(b.lastTerminalByBuild)
		}
		prev, seen := b.lastTerminalByBuild[session.BuildID]
		if seen && prev == phase {
			return notify.Notification{}, b.notifier, false
		}
		b.lastTerminalByBuild[session.BuildID] = phase
	}

	severity := notify.SeveritySuccess
	title := fmt.Sprintf("Report ready: %s", session.BuildID)
	body := fmt.Sprintf("Project %s in %s finished successfully.", session.ProjectID, session.EnvID)
	if phase == domain.PhaseFailed {
		severity = notify.SeverityError
		title = fmt.Sprintf("Upload failed: %s", session.BuildID)
		if session.Error != "" {
			body = session.Error
		} else {
			body = fmt.Sprintf("Project %s in %s failed during %s.", session.ProjectID, session.EnvID, session.FailedAtPhase)
		}
	}

	return notify.Notification{
		Title:    title,
		Body:     body,
		Category: "upload",
		Severity: severity,
		Payload: map[string]any{
			"session_id":       session.ID,
			"upload_id":        session.UploadID,
			"build_id":         session.BuildID,
			"project_id":       session.ProjectID,
			"environment_id":   session.EnvID,
			"phase":            session.Phase,
			"failed_at_phase":  session.FailedAtPhase,
			"report_url":       session.ReportURL,
			"received_chunks":  session.ReceivedChunks,
			"total_chunks":     session.TotalChunks,
			"completed_at":     session.CompletedAt,
			"generation_error": session.Error,
		},
	}, b.notifier, true
}
