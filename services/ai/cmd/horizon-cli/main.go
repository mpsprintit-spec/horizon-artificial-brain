package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
	"github.com/project-horizon/horizon-core/services/ai/evolution"
)

const memoryFile = "brain_memory.json"

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
	horizon.Execution.RegisterPlugin("terbang", &plugin.DronePlugin{})
	horizon.Execution.RegisterPlugin("logsystem", &plugin.ChatbotPlugin{})
	return &cli{
		horizon:    horizon,
		in:         bufio.NewScanner(input),
		out:        output,
		startedAt:  time.Now(),
		memoryPath: memoryPath,
		roleCatalog:     &evolution.RoleCatalog{},
		roleCatalogPath: "roles.json",
	}
}

func (c *cli) startup() error {
	c.printBanner()
	err := c.horizon.Knowledge.Load(c.memoryPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(c.out, "! Memory load skipped: %v\n", err)
	}
	fmt.Fprintln(c.out, "✓ Neural Memory")
	fmt.Fprintln(c.out, "✓ Activation Engine")
	fmt.Fprintln(c.out, "✓ Understanding Engine")
	fmt.Fprintln(c.out, "✓ Reasoning Engine")
	fmt.Fprintln(c.out, "✓ Language Engine")
	fmt.Fprintln(c.out, "✓ Learning Engine")
	fmt.Fprintln(c.out, "✓ WebSearch Engine")
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
	if err := c.horizon.Knowledge.Save(c.memoryPath); err != nil {
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
	signals, err := c.perceiver.Perceive(input)
	if err != nil || len(signals) == 0 {
		return pipelineResult{Input: input, Answer: "Input tidak dapat dipersepsi."}
	}
	tokens := signals[0].Tokens

	result := c.horizon.Pulse(ctx, core.TaskPulse{
		Stimulus: input,
		Context:  strings.Join(c.context, " "),
	})

	c.context = deriveContext(tokens, result.Concepts)
	return pipelineResult{
		Input:         input,
		Tokens:        tokens,
		Answer:        result.Answer,
		Confidence:    result.Confidence,
		Activation:    rankedTokens(c.horizon),
		Understanding: strings.Join(result.Concepts, ", "),
		Reasoning:     result.Answer,
		Context:       c.context,
		Hypothesis:    hypothesisSummary(result.Hypotheses),
		NeedsWeb:      result.NeedsWebSearch,
		Learned:       result.Learned,
		PatternEvidence: result.PatternEvidence,
		Intent:        result.Intent,
		Path:          result.Path,
		FocusToken:    result.FocusToken,
		FunctionalSignals: result.FunctionalSignals,
		InterpretationNotes: result.InterpretationNotes,
		Propositions: result.Propositions,
		Constraints: result.Constraints,
		EvidencePaths: result.EvidencePaths,
		EvalStatus: result.EvalStatus,
	}
}

func (c *cli) handleCommand(input string) bool {
	switch strings.ToLower(input) {
	case "help":
		fmt.Fprintln(c.out, "Commands: help, exit, quit, memory, nodes, synapses, activation, context, history, clear, save, load, debug on, debug off")
	case "exit", "quit":
		fmt.Fprintln(c.out, "Shutting down Horizon...")
	case "memory":
		fmt.Fprintf(c.out, "Memory: %d nodes, %d synapses\n", c.nodeCount(), c.synapseCount())
	case "":
	case "nodes":
		var tokens []string
		for _, n := range c.horizon.Knowledge.Registry.Nodes() {
			tokens = append(tokens, n.Token)
		}
		fmt.Fprintf(c.out, "Nodes (%d): %s\n", len(tokens), strings.Join(tokens, ", "))
	case "synapses":
		fmt.Fprintf(c.out, "Synapses: %d\n", c.synapseCount())
	case "activation":
		fmt.Fprintf(c.out, "Activation: %s\n", strings.Join(c.last.Activation, ", "))
	case "context":
		fmt.Fprintf(c.out, "Context: %s\n", strings.Join(c.context, ", "))
	case "history":
		fmt.Fprintln(c.out, strings.Join(c.history, "\n"))
	case "clear":
		fmt.Fprint(c.out, "\033[H\033[2J")
	case "save":
		if err := c.horizon.Knowledge.Save(c.memoryPath); err != nil {
			fmt.Fprintf(c.out, "Save failed: %v\n", err)
		} else {
			fmt.Fprintln(c.out, "Memory saved.")
		}
	case "load":
		if err := c.horizon.Knowledge.Load(c.memoryPath); err != nil {
			fmt.Fprintf(c.out, "Load failed: %v\n", err)
		} else {
			fmt.Fprintln(c.out, "Memory loaded.")
		}
	case "debug on":
		c.debug = true
		fmt.Fprintln(c.out, "Debug enabled.")
	case "clusters":
		clusters := evolution.ClusterNodes(c.horizon.Knowledge)
		for _, cl := range clusters {
			var words []string
			for _, id := range cl.Members {
				if n := c.horizon.Knowledge.Registry.GetByID(id); n != nil {
					words = append(words, n.Token)
				}
			}
			status := "belum cukup mapan"
			if len(cl.Members) >= evolution.StabilityThreshold {
				if name := c.roleCatalog.Match(cl.Centroid); name != "" {
					status = "peran: " + name
				} else {
					status = fmt.Sprintf("BARU, belum dikenal -- beri nama: name %d <nama>", cl.ID)
				}
			}
			fmt.Fprintf(c.out, "Cluster %d (%s): %s\n", cl.ID, status, strings.Join(words, ", "))
		}
	case "debug off":
		c.debug = false
		fmt.Fprintln(c.out, "Debug disabled.")
	default:
		if strings.HasPrefix(strings.ToLower(input),"stats ") {
			word := strings.TrimSpace(input[len("stats "):])
			node := c.horizon.Knowledge.Fetch(word)
			if node == nil {
				fmt.Fprintf(c.out, "Node %q belum ada.\n", word)
				return true
			}
			s := evolution.ComputeStats(c.horizon.Knowledge, node)
			fmt.Fprintf(c.out, "Stats %q: IsA-masuk=%d IsA-keluar=%d Deskriptif-masuk=%d JenisRelasi=%d Keluar=%d Masuk=%d RataConfidence=%.2f\n",
				word, s.IsAIncoming, s.IsAOutgoing, s.DescriptiveIncoming, s.DistinctRelationKinds, s.OutDegree, s.InDegree, s.AverageConfidence)
			return true
		}
		if strings.HasPrefix(strings.ToLower(input), "relations ") {
			word := strings.TrimSpace(input[len("relations "):])
			node := c.horizon.Knowledge.Fetch(word)
			if node == nil {
				fmt.Fprintf(c.out, "Node %q belum ada.\n", word)
				return true
			}
			fmt.Fprintf(c.out, "Keluar dari %q:\n", word)
			for _, s := range node.OutboundAll() {
			id := s.TargetID
				target := c.horizon.Knowledge.Registry.GetByID(id)
				name := "?"
				if target != nil {
					name = target.Token
				}
				fmt.Fprintf(c.out, "  --[%s]--> %s (weight=%.2f confidence=%.2f)\n", s.Kind, name, s.Weight, s.Confidence)
			}
			fmt.Fprintf(c.out, "Masuk ke %q:\n", word)
			for _, other := range c.horizon.Knowledge.Registry.Nodes() {
				if other.ID == node.ID {
					continue
				}
				if s := other.SynapsesTo(node.ID).Find("", false); false {
		_ = s
	} else if list := other.SynapsesTo(node.ID); len(list) > 0 {
		s := list[0]
					fmt.Fprintf(c.out, "  %s --[%s]--> (weight=%.2f confidence=%.2f)\n", other.Token, s.Kind, s.Weight, s.Confidence)
				}
			}
			return true
		}
		if strings.HasPrefix(strings.ToLower(input), "name ") {
			parts := strings.SplitN(strings.TrimSpace(input[len("name "):]), " ", 2)
			if len(parts) != 2 {
				fmt.Fprintln(c.out, "Pakai: name <id-cluster> <nama>")
				return true
			}
			var id int
			fmt.Sscanf(parts[0], "%d", &id)
			label := parts[1]
			clusters := evolution.ClusterNodes(c.horizon.Knowledge)
			for _, cl := range clusters {
				if cl.ID == id {
					if len(cl.Members) < evolution.StabilityThreshold {
						fmt.Fprintln(c.out, "Cluster ini belum cukup mapan untuk diberi nama.")
						return true
					}
					c.roleCatalog.Learn(label, cl.Centroid)
					fmt.Fprintf(c.out, "Dicatat: pola cluster %d sekarang dikenal sebagai %q.\n", id, label)
					return true
				}
			}
			fmt.Fprintln(c.out, "Cluster tidak ditemukan.")
			return true
		}
		if strings.ToLower(strings.TrimSpace(input)) == "patterns" {
			for _, ps := range c.horizon.Knowledge.Patterns.All() {
				var words []string
				for _, id := range ps.Members {
					if n := c.horizon.Knowledge.Registry.GetByID(id); n != nil {
						words = append(words, n.Token)
					}
				}
				resultWord := "?"
				if n := c.horizon.Knowledge.Registry.GetByID(ps.Result); n != nil {
					resultWord = n.Token
				}
				fmt.Fprintf(c.out, "{%s} -> %s (weight=%.2f confidence=%.2f freq=%d)\n",
					strings.Join(words, ", "), resultWord, ps.Weight, ps.Confidence, ps.Frequency)
			}
			return true
		}
		return false
	}
	return true
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
func (c *cli) nodeCount() int { return len(c.horizon.Knowledge.Registry.Nodes()) }
func (c *cli) synapseCount() int {
	total := 0
	for _, n := range c.horizon.Knowledge.Registry.Nodes() {
		total += len(n.Synapses)
	}
	return total
}

var contextNoise = map[string]bool{
	"itu": true, "ini": true, "adalah": true, "yang": true, "dan": true,
	"di": true, "ke": true, "dari": true, "untuk": true, "dengan": true,
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
func rankedTokens(h *core.HorizonEngine) []string {
	nodes := h.Thinking.LastState.ActivationHistory
	if len(nodes) == 0 {
		return nil
	}
	ranked := nodes[len(nodes)-1].RankedNodes
	out := make([]string, 0, len(ranked))
	for _, n := range ranked {
		out = append(out, n.Token)
	}
	sort.Strings(out)
	return out
}
func hypothesisSummary(hyps []thinking.Hypothesis) []string {
	out := make([]string, 0, len(hyps))
	for _, h := range hyps {
		out = append(out, fmt.Sprintf("confidence=%.2f evidence=%d conflicts=%d", h.Confidence, h.Evidence, h.Conflicts))
	}
	return out
}
