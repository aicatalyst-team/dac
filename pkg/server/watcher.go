package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors dashboard files for changes and broadcasts SSE events.
type Watcher struct {
	dir       string
	fsWatcher *fsnotify.Watcher
	clients   map[chan string]struct{}
	mu        sync.RWMutex
}

// NewWatcher creates a file watcher for the given directory.
func NewWatcher(dir string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating file watcher: %w", err)
	}

	if err := fw.Add(dir); err != nil {
		fw.Close()
		return nil, fmt.Errorf("watching directory %s: %w", dir, err)
	}

	return &Watcher{
		dir:       dir,
		fsWatcher: fw,
		clients:   make(map[chan string]struct{}),
	}, nil
}

// Run starts processing file system events. Call in a goroutine.
func (w *Watcher) Run() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				log.Printf("file changed: %s", event.Name)
				w.broadcast(map[string]string{
					"type": "full_reload",
					"file": event.Name,
				})
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) broadcast(data map[string]string) {
	msg, _ := json.Marshal(data)
	msgStr := string(msg)

	w.mu.RLock()
	defer w.mu.RUnlock()

	for ch := range w.clients {
		select {
		case ch <- msgStr:
		default:
			// Client not consuming fast enough, skip.
		}
	}
}

func (w *Watcher) subscribe() chan string {
	ch := make(chan string, 16)
	w.mu.Lock()
	w.clients[ch] = struct{}{}
	w.mu.Unlock()
	return ch
}

func (w *Watcher) unsubscribe(ch chan string) {
	w.mu.Lock()
	delete(w.clients, ch)
	w.mu.Unlock()
	close(ch)
}

// handleSSE serves Server-Sent Events for live reload.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if s.watcher == nil {
		http.Error(w, "file watcher not available", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := s.watcher.subscribe()
	defer s.watcher.unsubscribe(ch)

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
