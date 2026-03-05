package pipeline

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LogWriter writes build logs to a file and broadcasts to WebSocket subscribers.
type LogWriter struct {
	mu          sync.RWMutex
	file        *os.File
	subscribers map[string]chan []byte
	offset      int64
}

func NewLogWriter(filePath string) (*LogWriter, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	return &LogWriter{
		file:        f,
		subscribers: make(map[string]chan []byte),
	}, nil
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	n, err = lw.file.Write(p)
	lw.offset += int64(n)
	// Broadcast to all subscribers while holding the lock
	data := append([]byte{}, p...)
	for _, ch := range lw.subscribers {
		select {
		case ch <- data:
		default: // Drop if subscriber is slow
		}
	}
	lw.mu.Unlock()
	return
}

func (lw *LogWriter) WriteString(s string) {
	lw.Write([]byte(s))
}

func (lw *LogWriter) WriteStage(stage, status string) {
	ts := time.Now().Format("15:04:05")
	lw.WriteString(fmt.Sprintf("\n=== [%s] Stage: %s - %s ===\n", ts, stage, status))
}

func (lw *LogWriter) WriteLog(format string, args ...interface{}) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	lw.WriteString(fmt.Sprintf("[%s] %s\n", ts, msg))
}

func (lw *LogWriter) Subscribe(id string) chan []byte {
	ch := make(chan []byte, 256)
	lw.mu.Lock()
	lw.subscribers[id] = ch
	lw.mu.Unlock()
	return ch
}

func (lw *LogWriter) Unsubscribe(id string) {
	lw.mu.Lock()
	if ch, ok := lw.subscribers[id]; ok {
		close(ch)
		delete(lw.subscribers, id)
	}
	lw.mu.Unlock()
}

func (lw *LogWriter) Close() {
	lw.mu.Lock()
	for id, ch := range lw.subscribers {
		close(ch)
		delete(lw.subscribers, id)
	}
	lw.mu.Unlock()
	lw.file.Close()
}

func (lw *LogWriter) FilePath() string {
	return lw.file.Name()
}
