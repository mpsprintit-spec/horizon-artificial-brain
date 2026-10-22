package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
	"github.com/project-horizon/horizon-core/services/ai/evolution"
)

const memoryFile = "services/ai/brain_memory.json"

type cli struct {
	horizon    *core.HorizonEngine
	perceiver  perception.UserInputPerception
	in         *bufio.Scanner
	out        io.Writer
	debug      bool
	history    []string
	context    []string
	last       pipelineResult
	startedAt  time.Time
	memoryPath string
	roleCatalog     *evolution.RoleCatalog
	roleCatalogPath string
	brainLive bool
	brainStop chan struct{}
	brainWG sync.WaitGroup
}

type pipelineResult struct {
	Input         string
	Tokens        []string
	Answer        string
	Confidence    float64
	Activation    []string
	Understanding string
	Reasoning     string
	Context       []string
	Hypothesis    []string
	NeedsWeb      bool
	Learned       bool
	PatternEvidence []string
	Intent        string
	Path          string
	FocusToken    string
	FunctionalSignals []string
	InterpretationNotes []string
	Propositions []string
	Constraints []string
	EvidencePaths []string
	EvalStatus string
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
		horizon:    horizon,
		in:         bufio.NewScanner(input),
		out:        &cliOutput{writer: output},
		startedAt:  time.Now(),
		memoryPath: memoryPath,
		roleCatalog:     &evolution.RoleCatalog{},
		roleCatalogPath: "roles.json",
		brainStop: make(chan struct{}),
	}
}

func (c *cli) startup() error {
	c.printBanner()
	err := c.horizon.LoadActiveBrain(c.memoryPath)
	if os.IsNotExist(err) {
		// One-time migration path for older local checkouts. The canonical
		// runtime store remains services/ai/brain_memory.json.
		if legacyErr := c.horizon.LoadActiveBrain("brain_memory.json"); legacyErr == nil {
			if saveErr := c.horizon.SaveActiveBrain(c.memoryPath); saveErr != nil {
				fmt.Fprintf(c.out, "! Memory migration save skipped: %v\n", saveErr)
			}
			err = nil
		}
	}
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(c.out, "! Memory load skipped: %v\n", err)
	}
	if err == nil {
		if _, activationErr := c.horizon.ActivateActiveBootstrap(); activationErr != nil {
			fmt.Fprintf(c.out, "! Bootstrap activation skipped: %v\n", activationErr)
		}
	}
	fmt.Fprintln(c.out, "✓ Neural Memory")
	fmt.Fprintln(c.out, "✓ Activation Engine")
	fmt.Fprintln(c.out, "✓ Understanding Engine")
	fmt.Fprintln(c.out, "✓ Reasoning Engine")
	fmt.Fprintln(c.out, "✓ Language Engine")
	fmt.Fprintln(c.out, "✓ Learning Engine")
	fmt.Fprintln(c.out, "✓ WebSearch Engine")
	fmt.Fprintln(c.out, "✓ Brain Monitor: starting autonomous loop")
	c.startBrainMonitor()
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	if catalog, cerr := evolution.LoadRoleCatalog(c.roleCatalogPath); cerr == nil {
		c.roleCatalog = catalog
	}
	return err
}

func (c *cli) printBanner() {
	fmt.Fprintln(c.out, "======================================")
	fmt.Fprintln(c.out, " Horizon Cognitive Intelligence")
	fmt.Fprintln(c.out, " Neural Semantic Memory Ready")
	fmt.Fprintln(c.out, "======================================")
}

func (c *cli) run(ctx context.Context) {
	for {
		fmt.Fprint(c.out, "> ")
		if !c.in.Scan() {
			break
		}
		input := strings.TrimSpace(c.in.Text())
		if input == "" {
			continue
		}
		if c.handleCommand(input) {
			if input == "exit" || input == "quit" {
				break
			}
			continue
		}
		result := c.process(ctx, input)
		c.last = result
		c.history = append(c.history, input)
		if len(c.history) > 25 {
			c.history = c.history[len(c.history)-25:]
		}
		if c.debug {
			c.printDebug(result)
		}
		fmt.Fprintln(c.out, result.Answer)
	}
	c.stopBrainMonitor()
	if err := c.horizon.SaveActiveBrain(c.memoryPath); err != nil {
		fmt.Fprintf(c.out, "Shutdown save warning: %v\n", err)
	}
	if err := c.roleCatalog.Save(c.roleCatalogPath); err != nil {
		fmt.Fprintf(c.out, "Role catalog save warning: %v\n", err)
	}
	c.printSessionStats()
}

// process sekarang TIPIS: cuma persepsi input, lalu serahkan semua keputusan
// (kapan belajar, kapan cari web, mode ingat atau tidak) ke horizon.Pulse --
// satu-satunya jalur resmi otak Horizon.
func (c *cli) process(ctx context.Context, input string) pipelineResult {
	_ = ctx // canonical Pulse owns the cognitive pipeline and does not accept a legacy context argument.
	result, concepts, err := c.horizon.PulseWithLabels(input)
	if err != nil {
		return pipelineResult{Input: input, Answer: "Input tidak dapat diproses: " + err.Error()}
	}

	c.context = deriveContext(nil, concepts)
	answer := strings.Join(concepts, ", ")
	if answer == "" {
		answer = "Tidak ada node neural yang teraktivasi."
	}
	return pipelineResult{
		Input: input,
		Tokens: strings.Fields(input),
		Answer: answer,
		Path: "neural_runtime",
		Confidence: result.Answer.Confidence,
		Activation: activeNodeTokens(result.State.ActiveNodeIDs),
		Understanding: strings.Join(concepts, ", "),
		Reasoning: answer,
		Context: c.context,
		FunctionalSignals: nil,
		InterpretationNotes: []string{"source=" + result.Interpretation.Source},
		Propositions: nil,
		Constraints: nil,
		EvidencePaths: nil,
		EvalStatus: "cognitive_interpretation",
	}
}

// handleCommand never exposes a live neural pointer to inspection code.
// Save/load and autonomous monitor controls are separate managed mutations.
func (c *cli) handleCommand(input string) bool {
	lower:=strings.ToLower(strings.TrimSpace(input))
	switch lower {
	case "brain":
		c.toggleBrainMonitor()
		return true
	case "save":
		if err:=c.horizon.SaveActiveBrain(c.memoryPath);err!=nil {
			fmt.Fprintf(c.out,"Save failed: %v\n",err)
		} else {fmt.Fprintln(c.out,"Memory saved.")}
		return true
	case "load":
		if err:=c.horizon.LoadActiveBrain(c.memoryPath);err!=nil {
			fmt.Fprintf(c.out,"Load failed: %v\n",err)
		} else {fmt.Fprintln(c.out,"Memory loaded.")}
		return true
	case "help","exit","quit","activation","context","history","clear","debug on","debug off":
		// These commands only inspect local CLI state, so they must not
		// serialize a potentially massive neural graph.
		return c.handleCommandWithSnapshot(input,knowledge.BrainReadSnapshot{})
	case "memory","nodes","synapses","clusters","patterns":
		// Fall through to the managed brain inspection below.
	default:
		if !strings.HasPrefix(lower,"stats ") &&
			!strings.HasPrefix(lower,"relations ") &&
			!strings.HasPrefix(lower,"name ") {
			// The ordinary user input path must not construct a neural
			// snapshot before entering the cognition pipeline.
			return false
		}
	}
	inspection,err:=c.horizon.InspectActiveBrain()
	if err!=nil {
		fmt.Fprintf(c.out,"Brain command unavailable: %v\n",err)
		return true
	}
	return c.handleCommandWithSnapshot(input,inspection.Graph)
}

func (c *cli) handleCommandWithSnapshot(input string,brain knowledge.BrainReadSnapshot) bool {
	units:=brain.NeuralUnits()
	edges:=brain.Synapses()
	switch strings.ToLower(input) {
	case "help":
		fmt.Fprintln(c.out,"Commands: help, exit, quit, brain, memory, nodes, synapses, activation, context, history, clear, save, load, debug on, debug off")
	case "exit","quit":
		fmt.Fprintln(c.out,"Shutting down Horizon...")
	case "memory":
		fmt.Fprintf(c.out,"Memory: %d nodes, %d synapses\n",len(units),len(edges))
	case "":
	case "nodes":
		var tokens []string
		for _,n:=range units {tokens=append(tokens,fmt.Sprintf("node:%d",n.ID))}
		fmt.Fprintf(c.out,"Nodes (%d): %s\n",len(tokens),strings.Join(tokens,", "))
	case "synapses":
		fmt.Fprintf(c.out,"Synapses: %d\n",len(edges))
	case "activation":
		fmt.Fprintf(c.out,"Activation: %s\n",strings.Join(c.last.Activation,", "))
	case "context":
		fmt.Fprintf(c.out,"Context: %s\n",strings.Join(c.context,", "))
	case "history":
		fmt.Fprintln(c.out,strings.Join(c.history,"\n"))
	case "clear":
		fmt.Fprint(c.out,"\x1b[H\x1b[2J")
	case "debug on":
		c.debug=true
		fmt.Fprintln(c.out,"Debug enabled.")
	case "debug off":
		c.debug=false
		fmt.Fprintln(c.out,"Debug disabled.")
	case "clusters":
		for _,cl:=range evolution.ClusterSnapshot(brain) {
			var words []string
			for _,id:=range cl.Members {
				if brain.HasNode(id) {words=append(words,fmt.Sprintf("node:%d",id))}
			}
			status:="belum cukup mapan"
			if len(cl.Members)>=evolution.StabilityThreshold {
				if name:=c.roleCatalog.Match(cl.Centroid);name!="" {
					status="peran: "+name
				} else {
					status=fmt.Sprintf("BARU, belum dikenal -- beri nama: name %d <nama>",cl.ID)
				}
			}
			fmt.Fprintf(c.out,"Cluster %d (%s): %s\n",cl.ID,status,strings.Join(words,", "))
		}
	default:
		lower:=strings.ToLower(strings.TrimSpace(input))
		if strings.HasPrefix(lower,"stats ") {
			word:=strings.TrimSpace(input[len("stats "):])
			nodeID,ok:=brain.NodeForSurface(word)
			if !ok {
				fmt.Fprintf(c.out,"Node %q belum ada.\n",word)
				return true
			}
			s:=evolution.ComputeSnapshotStats(brain,nodeID)
			fmt.Fprintf(c.out,"Stats %q: IsA-masuk=%d IsA-keluar=%d Deskriptif-masuk=%d JenisRelasi=%d Keluar=%d Masuk=%d RataConfidence=%.2f\n",
				word,s.IsAIncoming,s.IsAOutgoing,s.DescriptiveIncoming,s.DistinctRelationKinds,s.OutDegree,s.InDegree,s.AverageConfidence)
			return true
		}
		if strings.HasPrefix(lower,"relations ") {
			word:=strings.TrimSpace(input[len("relations "):])
			nodeID,ok:=brain.NodeForSurface(word)
			if !ok {
				fmt.Fprintf(c.out,"Node %q belum ada.\n",word)
				return true
			}
			fmt.Fprintf(c.out,"Keluar dari %q:\n",word)
			for _,edge:=range edges {
				if edge.SourceID!=nodeID {continue}
				name:="?"
				if brain.HasNode(edge.TargetID) {name=fmt.Sprintf("node:%d",edge.TargetID)}
				fmt.Fprintf(c.out,"  --[%s]--> %s (weight=%.2f confidence=%.2f)\n",edge.Kind,name,edge.Weight,edge.Confidence)
			}
			fmt.Fprintf(c.out,"Masuk ke %q:\n",word)
			for _,edge:=range edges {
				if edge.TargetID==nodeID && edge.SourceID!=nodeID {
					fmt.Fprintf(c.out,"  node:%d --[%s]--> (weight=%.2f confidence=%.2f)\n",
						edge.SourceID,edge.Kind,edge.Weight,edge.Confidence)
				}
			}
			return true
		}
		if strings.HasPrefix(lower,"name ") {
			parts:=strings.SplitN(strings.TrimSpace(input[len("name "):])," ",2)
			if len(parts)!=2 {
				fmt.Fprintln(c.out,"Pakai: name <id-cluster> <nama>")
				return true
			}
			var id int
			fmt.Sscanf(parts[0],"%d",&id)
			label:=parts[1]
			for _,cl:=range evolution.ClusterSnapshot(brain) {
				if cl.ID!=id {continue}
				if len(cl.Members)<evolution.StabilityThreshold {
					fmt.Fprintln(c.out,"Cluster ini belum cukup mapan untuk diberi nama.")
					return true
				}
				c.roleCatalog.Learn(label,cl.Centroid)
				fmt.Fprintf(c.out,"Dicatat: pola cluster %d sekarang dikenal sebagai %q.\n",id,label)
				return true
			}
			fmt.Fprintln(c.out,"Cluster tidak ditemukan.")
			return true
		}
		if lower=="patterns" {
			for _,ps:=range brain.Patterns() {
				var words []string
				for _,id:=range ps.Members {
					if brain.HasNode(id) {words=append(words,fmt.Sprintf("node:%d",id))}
				}
				resultWord:="?"
				if brain.HasNode(ps.Result) {resultWord=fmt.Sprintf("node:%d",ps.Result)}
				fmt.Fprintf(c.out,"{%s} -> %s (weight=%.2f confidence=%.2f freq=%d)\n",
					strings.Join(words,", "),resultWord,ps.Weight,ps.Confidence,ps.Frequency)
			}
			return true
		}
		return false
	}
	return true
}


func (c *cli) toggleBrainMonitor() {
	if c.brainLive {
		c.stopBrainMonitor()
		c.safePrintln("[BRAIN] autonomous loop OFF")
		return
	}
	c.startBrainMonitor()
	c.safePrintln("[BRAIN] autonomous loop ON")
}

func (c *cli) startBrainMonitor() {
	if c.brainLive {
		return
	}
	c.brainLive = true
	c.brainStop = make(chan struct{})
	c.brainWG.Add(1)
	go c.runBrainMonitor(c.brainStop)
}

func (c *cli) stopBrainMonitor() {
	if !c.brainLive {
		return
	}
	c.brainLive = false
	close(c.brainStop)
	c.brainWG.Wait()
}

func (c *cli) runBrainMonitor(stop <-chan struct{}) {
	defer c.brainWG.Done()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			output, err := c.horizon.ThinkActive(1)
			if err != nil {
				c.safePrintf("[BRAIN] error: %v\n", err)
				if errors.Is(err, runtime.ErrRuntimeRecoveryRequired) || errors.Is(err, runtime.ErrRuntimeSupervisorClosed) {
					return
				}
				continue
			}
			c.printBrainState(output)
		}
	}
}

func (c *cli) printBrainState(output runtime.CognitiveOutput) {
	ids := append([]knowledge.NodeID(nil), output.RankedNodeIDs...)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("node:%d A=%.3f C=%.3f", id, output.Activations[id], output.Confidence[id]))
	}
	predIDs := make([]knowledge.NodeID, 0, len(output.Prediction.State))
	for id, value := range output.Prediction.State {
		if value > 0.01 {
			predIDs = append(predIDs, id)
		}
	}
	sort.Slice(predIDs, func(i, j int) bool {
		return output.Prediction.State[predIDs[i]] > output.Prediction.State[predIDs[j]]
	})
	predParts := make([]string, 0, len(predIDs))
	for _, id := range predIDs {
		predParts = append(predParts, fmt.Sprintf("node:%d %.3f", id, output.Prediction.State[id]))
	}
	if len(parts) == 0 {
		parts = append(parts, "QUIESCENT")
	}
	if len(predParts) == 0 {
		predParts = append(predParts, "none")
	}
	c.safePrintf("[BRAIN] seq=%d resonance=%.3f error=%.3f active=%s | prediction=%s\n",
		output.Sequence, output.Resonance, output.PredictionError,
		strings.Join(parts, " ; "), strings.Join(predParts, " ; "))
}

func (c *cli) safePrintf(format string, args ...interface{}) {
	fmt.Fprintf(c.out, format, args...)
}

func (c *cli) safePrintln(value string) {
	fmt.Fprintln(c.out, value)
}

func (c *cli) printDebug(r pipelineResult) {
	fmt.Fprintf(c.out, "[debug] Activation: %s\n", strings.Join(r.Activation, ", "))
	fmt.Fprintf(c.out, "[debug] Understanding: %s\n", r.Understanding)
	fmt.Fprintf(c.out, "[debug] Reasoning: %s\n", r.Reasoning)
	fmt.Fprintf(c.out, "[debug] Confidence: %.2f\n", r.Confidence)
	fmt.Fprintf(c.out, "[debug] Context: %s\n", strings.Join(r.Context, ", "))
	fmt.Fprintf(c.out, "[debug] Hypothesis: %s\n", strings.Join(r.Hypothesis, ", "))
	fmt.Fprintf(c.out, "[debug] WebSearch: %t\n", r.NeedsWeb)
	fmt.Fprintf(c.out, "[debug] Learning: %t\n", r.Learned)
	fmt.Fprintf(c.out, "[debug] PatternEvidence: %s\n", strings.Join(r.PatternEvidence, " | "))
	fmt.Fprintf(c.out, "[debug] Intent: %s\n", r.Intent)
	fmt.Fprintf(c.out, "[debug] Path: %s\n", r.Path)
	fmt.Fprintf(c.out, "[debug] Focus: %s\n", r.FocusToken)
	fmt.Fprintf(c.out, "[debug] FunctionalSignals: %s\n", strings.Join(r.FunctionalSignals, " | "))
	fmt.Fprintf(c.out, "[debug] InterpretationNotes: %s\n", strings.Join(r.InterpretationNotes, " | "))
	fmt.Fprintf(c.out, "[debug] Proposition: %s\n", strings.Join(r.Propositions, " | "))
	fmt.Fprintf(c.out, "[debug] Constraints: %s\n", strings.Join(r.Constraints, " | "))
	fmt.Fprintf(c.out, "[debug] EvidencePaths: %s\n", strings.Join(r.EvidencePaths, " | "))
	fmt.Fprintf(c.out, "[debug] Decision: %s\n", r.EvalStatus)
}

func (c *cli) printSessionStats() {
	fmt.Fprintf(c.out, "Session: %d inputs, %d nodes, %d synapses, %s uptime\n", len(c.history), c.nodeCount(), c.synapseCount(), time.Since(c.startedAt).Round(time.Millisecond))
}
func (c *cli) nodeCount() int {
	inspection,err:=c.horizon.InspectActiveBrain()
	if err!=nil {return 0}
	nodes,_,_:=inspection.Graph.Counts()
	return nodes
}
func (c *cli) synapseCount() int {
	inspection,err:=c.horizon.InspectActiveBrain()
	if err!=nil {return 0}
	_,synapses,_:=inspection.Graph.Counts()
	return synapses
}

func activeNodeTokens(ids []knowledge.NodeID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids { out = append(out, fmt.Sprintf("node:%d", id)) }
	sort.Strings(out)
	return out
}

var contextNoise = map[string]bool{
	"itu": true, "ini": true, "adalah": true, "yang": true, "dan": true,
	"di": true, "ke": true, "dari": true, "untuk": true, "dengan": true,
}

func containsString(values []string, target string) bool {
	for _, value := range values { if value == target { return true } }
	return false
}

func deriveContext(tokens, concepts []string) []string {
	out := append([]string{}, concepts...)
	if len(out) == 0 {
		for _, t := range tokens {
			if !contextNoise[t] {
				out = append(out, t)
			}
		}
	}
	if len(out) > 5 {
		return out[:5]
	}
	return out
}
func hypothesisSummary(hyps []thinking.Hypothesis) []string {
	out := make([]string, 0, len(hyps))
	for _, h := range hyps {
		out = append(out, fmt.Sprintf("confidence=%.2f evidence=%d conflicts=%d", h.Confidence, h.Evidence, h.Conflicts))
	}
	return out
}
