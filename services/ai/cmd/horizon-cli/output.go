package main

import (
	"io"
	"sync"
)

// cliOutput serializes every REPL and autonomous-monitor write, including
// command paths that use fmt directly rather than the monitor print helpers.
type cliOutput struct {
	mu     sync.Mutex
	writer io.Writer
}

func (w *cliOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}
