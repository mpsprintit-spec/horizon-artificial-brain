package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestCLICommandAndMonitorOutputShareWriterLock(t *testing.T) {
	var output bytes.Buffer
	app := newCLI(strings.NewReader(""), &output, filepath.Join(t.TempDir(), "brain.json"))
	const lines = 100
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < lines; i++ {
			app.printBrainState(runtime.CognitiveOutput{})
		}
	}()
	close(start)
	for i := 0; i < lines; i++ {
		app.handleCommand("help")
	}
	wg.Wait()
	text := output.String()
	if strings.Count(text, "Commands:") != lines || strings.Count(text, "[BRAIN] seq=") != lines || strings.Count(text, "\n") != 2*lines {
		t.Fatal("concurrent command and monitor output lost or interleaved records")
	}
}
