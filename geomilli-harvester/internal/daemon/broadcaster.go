package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
	recentLogs  []LogEvent
	maxLogs     int
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan []byte]struct{}),
		recentLogs:  make([]LogEvent, 0, 100),
		maxLogs:     100,
	}
}

func (b *Broadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 100)
	b.subscribers[ch] = struct{}{}
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, ch)
	close(ch)
}

func (b *Broadcaster) Broadcast(event string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(data)))

	b.mu.Lock()
	defer b.mu.Unlock()

	if event == "log" {
		if logEv, ok := payload.(LogEvent); ok {
			b.recentLogs = append(b.recentLogs, logEv)
			if len(b.recentLogs) > b.maxLogs {
				b.recentLogs = b.recentLogs[len(b.recentLogs)-b.maxLogs:]
			}
		}
	}

	for ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			// Non-blocking drop if client is too slow
		}
	}
}

func (b *Broadcaster) BroadcastLog(level, message string) {
	b.Broadcast("log", LogEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	})
}

func (b *Broadcaster) BroadcastProgress(p ProgressEvent) {
	if p.Timestamp == "" {
		p.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	b.Broadcast("progress", p)
}

func (b *Broadcaster) BroadcastQueue(q QueueInfo) {
	b.Broadcast("queue", q)
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Send initial ping and log history to newly connected client
	fmt.Fprintf(w, ": connected\n\n")
	b.mu.RLock()
	for _, l := range b.recentLogs {
		data, err := json.Marshal(l)
		if err == nil {
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(data))
		}
	}
	b.mu.RUnlock()
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			// Heartbeat
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
