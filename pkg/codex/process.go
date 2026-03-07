package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Process manages a codex app-server subprocess via stdio JSON-RPC.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	writeMu sync.Mutex
	nextID  atomic.Int64
	pending sync.Map // int64 -> chan *Message

	subsMu sync.RWMutex
	subs   map[string][]chan *Event // threadID -> subscriber channels

	// tracks which threads have received their first turn (for system prompt injection)
	initialized sync.Map // threadID -> bool

	started bool
	startMu sync.Mutex

	// Session event logging
	logDir   string
	logMu    sync.Mutex
	logFiles map[string]*os.File
}

// Message is a JSON-RPC 2.0 message on the wire.
type Message struct {
	Method string          `json:"method,omitempty"`
	ID     *int64          `json:"id,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// Event is a notification forwarded to SSE subscribers.
type Event struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// New creates a process manager. The subprocess is started lazily.
// logDir specifies where to write per-session JSONL event logs. Empty disables logging.
func New(logDir string) *Process {
	return &Process{
		subs:     make(map[string][]chan *Event),
		logDir:   logDir,
		logFiles: make(map[string]*os.File),
	}
}

// ensureStarted starts the codex process if not already running.
func (p *Process) ensureStarted() error {
	p.startMu.Lock()
	defer p.startMu.Unlock()

	if p.started {
		return nil
	}

	cmd := exec.Command("codex", "app-server")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting codex app-server: %w", err)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.started = true

	go p.readLoop()

	// Perform JSON-RPC initialization handshake.
	if err := p.initialize(); err != nil {
		p.kill()
		return fmt.Errorf("codex handshake failed: %w", err)
	}

	log.Printf("codex app-server started (pid %d)", cmd.Process.Pid)
	return nil
}

func (p *Process) initialize() error {
	_, err := p.call("initialize", json.RawMessage(`{
		"clientInfo": {
			"name": "dac",
			"title": "DAC Dashboard Editor",
			"version": "1.0.0"
		},
		"capabilities": {
			"experimentalApi": true,
			"optOutNotificationMethods": []
		}
	}`))
	if err != nil {
		return err
	}
	p.notify("initialized", json.RawMessage(`{}`))
	time.Sleep(500 * time.Millisecond) // allow codex to finish post-init setup
	return nil
}

// call sends a JSON-RPC request and waits for the response.
func (p *Process) call(method string, params json.RawMessage) (*Message, error) {
	id := p.nextID.Add(1)
	ch := make(chan *Message, 1)
	p.pending.Store(id, ch)
	defer p.pending.Delete(id)

	msg := Message{Method: method, ID: &id, Params: params}
	if err := p.send(&msg); err != nil {
		return nil, err
	}

	resp, ok := <-ch
	if !ok || resp == nil {
		return nil, fmt.Errorf("codex process closed")
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp, nil
}

// notify sends a JSON-RPC notification (fire-and-forget).
func (p *Process) notify(method string, params json.RawMessage) {
	msg := Message{Method: method, Params: params}
	p.send(&msg) //nolint:errcheck
}

func (p *Process) send(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_, err = fmt.Fprintf(p.stdin, "%s\n", data)
	return err
}

// readLoop reads JSONL from stdout and routes messages.
func (p *Process) readLoop() {
	reader := bufio.NewReaderSize(p.stdout, 10*1024*1024)

	log.Printf("codex: readLoop started")
	lineCount := 0
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("codex: read error after %d lines: %v", lineCount, err)
			} else {
				log.Printf("codex: EOF after %d lines", lineCount)
			}
			break
		}

		lineCount++
		line = line[:len(line)-1] // trim newline
		if len(line) == 0 {
			continue
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("codex: parse error: %v (line: %.200s)", err, string(line))
			continue
		}

		if msg.Method != "" {
			if msg.Method == "codex/event/warning" || msg.Method == "codex/event/error" || msg.Method == "turn/completed" {
				log.Printf("codex event: %s params: %s", msg.Method, string(msg.Params))
			} else {
				log.Printf("codex event: %s", msg.Method)
			}
		}
		p.route(&msg)
	}

	// Process exited — fail all pending requests.
	p.pending.Range(func(key, val any) bool {
		close(val.(chan *Message))
		p.pending.Delete(key)
		return true
	})

	p.startMu.Lock()
	p.started = false
	p.startMu.Unlock()
	log.Printf("codex app-server exited")
}

func (p *Process) route(msg *Message) {
	// Response to one of our requests.
	if msg.ID != nil && msg.Method == "" {
		if ch, ok := p.pending.Load(*msg.ID); ok {
			ch.(chan *Message) <- msg
		}
		return
	}

	// Server-initiated request (approval) — auto-approve.
	if msg.ID != nil && msg.Method != "" {
		p.handleApproval(msg)
		return
	}

	// Notification — forward to thread subscribers.
	if msg.Method != "" {
		p.forwardEvent(msg)
	}
}

func (p *Process) handleApproval(msg *Message) {
	id := *msg.ID
	resp := Message{
		ID:     &id,
		Result: json.RawMessage(`"accept"`),
	}
	if err := p.send(&resp); err != nil {
		log.Printf("codex: failed to send approval: %v", err)
	}
}

func (p *Process) forwardEvent(msg *Message) {
	threadID := extractThreadID(msg.Params)
	if threadID == "" {
		log.Printf("codex: no threadID for event %s", msg.Method)
		return
	}

	event := &Event{Method: msg.Method, Params: msg.Params}

	p.logEvent(threadID, msg)

	p.subsMu.RLock()
	subs := p.subs[threadID]
	p.subsMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// logEvent appends a raw event to the thread's JSONL log file.
func (p *Process) logEvent(threadID string, msg *Message) {
	if p.logDir == "" {
		return
	}

	p.logMu.Lock()
	defer p.logMu.Unlock()

	f, ok := p.logFiles[threadID]
	if !ok {
		if err := os.MkdirAll(p.logDir, 0o755); err != nil {
			log.Printf("codex: failed to create log dir: %v", err)
			return
		}
		ts := time.Now().Format("20060102-150405")
		path := filepath.Join(p.logDir, fmt.Sprintf("%s_%s.jsonl", ts, threadID[:min(8, len(threadID))]))
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("codex: failed to open log file: %v", err)
			return
		}
		p.logFiles[threadID] = f
		log.Printf("codex: logging session %s to %s", threadID, path)
	}

	entry := map[string]any{
		"ts":     time.Now().Format(time.RFC3339Nano),
		"method": msg.Method,
		"params": json.RawMessage(msg.Params),
	}
	data, _ := json.Marshal(entry)
	f.Write(data)
	f.Write([]byte("\n"))
}

// extractThreadID tries to find a threadId in the notification params.
func extractThreadID(params json.RawMessage) string {
	var p struct {
		ThreadID string `json:"threadId"`
		Thread   struct {
			ID string `json:"id"`
		} `json:"thread"`
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	json.Unmarshal(params, &p) //nolint:errcheck

	if p.ThreadID != "" {
		return p.ThreadID
	}
	if p.Thread.ID != "" {
		return p.Thread.ID
	}
	return ""
}

// StartThread creates a new codex thread.
func (p *Process) StartThread(cwd, model string) (string, error) {
	if err := p.ensureStarted(); err != nil {
		return "", err
	}

	if model == "" {
		model = "gpt-5.4"
	}

	params := map[string]any{
		"model":          model,
		"cwd":            cwd,
		"approvalPolicy": "never",
		"sandbox":        "danger-full-access",
	}
	data, _ := json.Marshal(params)

	resp, err := p.call("thread/start", data)
	if err != nil {
		return "", fmt.Errorf("starting thread: %w", err)
	}

	var result struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parsing thread response: %w", err)
	}
	return result.Thread.ID, nil
}

// StartTurn sends a user message to an existing thread.
func (p *Process) StartTurn(threadID string, input []map[string]any) error {
	if err := p.ensureStarted(); err != nil {
		return err
	}

	params := map[string]any{
		"threadId": threadID,
		"input":    input,
	}
	data, _ := json.Marshal(params)

	_, err := p.call("turn/start", data)
	return err
}

// InterruptTurn cancels the active turn.
func (p *Process) InterruptTurn(threadID, turnID string) error {
	params := map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	}
	data, _ := json.Marshal(params)
	_, err := p.call("turn/interrupt", data)
	return err
}

// Subscribe returns a channel that receives events for the given thread.
func (p *Process) Subscribe(threadID string) chan *Event {
	ch := make(chan *Event, 128)
	p.subsMu.Lock()
	p.subs[threadID] = append(p.subs[threadID], ch)
	p.subsMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel for a thread.
func (p *Process) Unsubscribe(threadID string, ch chan *Event) {
	p.subsMu.Lock()
	subs := p.subs[threadID]
	for i, s := range subs {
		if s == ch {
			p.subs[threadID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(p.subs[threadID]) == 0 {
		delete(p.subs, threadID)
	}
	p.subsMu.Unlock()
}

// IsThreadInitialized returns true if the thread has had its first turn.
func (p *Process) IsThreadInitialized(threadID string) bool {
	_, ok := p.initialized.Load(threadID)
	return ok
}

// MarkThreadInitialized marks a thread as having received its first turn.
func (p *Process) MarkThreadInitialized(threadID string) {
	p.initialized.Store(threadID, true)
}

func (p *Process) kill() {
	if p.stdin != nil {
		p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	p.startMu.Lock()
	p.started = false
	p.startMu.Unlock()
}

// Close shuts down the codex process and flushes log files.
func (p *Process) Close() error {
	p.kill()
	p.logMu.Lock()
	for _, f := range p.logFiles {
		f.Close()
	}
	p.logFiles = make(map[string]*os.File)
	p.logMu.Unlock()
	if p.cmd != nil {
		return p.cmd.Wait()
	}
	return nil
}
