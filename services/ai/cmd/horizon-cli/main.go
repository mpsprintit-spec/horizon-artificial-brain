package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/evolution"
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
)

const memoryFile = "brain_memory.json"

// cli is the presentation shell around the single Horizon brain.
type cli struct {
	horizon         *core.HorizonEngine
	in              *bufio.Scanner
	out             io.Writer
	startedAt       time.Time
	memoryPath      string
	roleCatalog     *evolution.RoleCatalog
	roleCatalogPath string
}

func main() {
	app := newCLI(os.Stdin, os.Stdout, memoryFile)
	if err := app.startup(); err != nil {
		fmt.Fprintf(app.out, "Startup warning: %v\n", err)
	}
	app.run(context.Background())
}

func newCLI(input io.Reader, output io.Writer, memoryPath string) *cli {
	horizon := core.NewHorizonEngine()
	horizon.WebSearch = websearch.NewEngine(websearch.WikipediaSearcher{})
	horizon.Gateway.RegisterPlugin("terbang", &plugin.DronePlugin{})
	horizon.Gateway.RegisterPlugin("logsystem", &plugin.ChatbotPlugin{})
	return &cli{
		horizon:         horizon,
		in:              bufio.NewScanner(input),
		out:             output,
		startedAt:       time.Now(),
		memoryPath:      memoryPath,
		roleCatalog:     &evolution.RoleCatalog{},
		roleCatalogPath: "roles.json",
	}
}
