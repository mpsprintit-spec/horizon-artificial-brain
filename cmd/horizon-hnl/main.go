// horizon-hnl is an offline, single-session HNL/1 research harness. It does not
// connect to a model provider, sensor, exchange or the production Brain.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/teaching"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	session := flag.String("session", "", "isolated session directory")
	init := flag.Bool("init", false, "create a fresh V5 research Brain; directory must not exist")
	request := flag.String("request", "", "one binary .hnl packet")
	reply := flag.String("reply", "", "output binary .hnl reply")
	inspect := flag.Bool("inspect", false, "decode request as diagnostic JSON without opening a Brain")
	clockNS := flag.Int64("clock-ns", 0, "trusted replay clock in Unix nanoseconds; default is wall time")
	flag.Parse()
	now := time.Now().UTC()
	if *clockNS != 0 {
		now = time.Unix(0, *clockNS).UTC()
	}
	var raw []byte
	if *request != "" {
		f, err := os.Open(*request)
		if err != nil {
			return err
		}
		raw, err = io.ReadAll(io.LimitReader(f, teaching.MaxPacketBytes+1))
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if _, err = teaching.Unmarshal(raw); err != nil {
			return err
		}
	}
	if *inspect {
		if len(raw) == 0 {
			return errors.New("inspect requires -request")
		}
		p, err := teaching.Unmarshal(raw)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}
	if *session == "" || (*init && len(raw) != 0) || (!*init && (len(raw) == 0 || *reply == "")) {
		return errors.New("use -session DIR -init, or -session DIR -request FILE -reply FILE; optional -clock-ns for a trusted simulator")
	}
	state := filepath.Join(*session, "brain.json")
	if *init {
		if err := os.Mkdir(*session, 0700); err != nil {
			return err
		}
	}
	log, err := brainruntime.OpenEventLog(filepath.Join(*session, "events.jsonl"))
	if err != nil {
		return err
	}
	defer log.Close()
	if *init {
		r := brainruntime.NewBrainRuntime(knowledge.NewBrain())
		r.SetEventLog(log)
		r.SetClock(brainruntime.NewFixedClock(now))
		plan, err := r.PlanNeuralWrittenGrounding()
		if err != nil {
			return err
		}
		if err = r.ApplyNeuralWrittenGrounding(plan, filepath.Join(*session, "before-written.json"), now); err != nil {
			return err
		}
		cfg := knowledge.DefaultCompositionConfig()
		cfg.Version = knowledge.NeuralCompositionResolutionV5
		cfg.Initialization = "random"
		c, err := r.PlanNeuralComposition(cfg)
		if err != nil {
			return err
		}
		if err = r.ApplyNeuralComposition(c, filepath.Join(*session, "before-composition.json"), now); err != nil {
			return err
		}
		return r.Checkpoint(state)
	}
	r, err := brainruntime.RebuildRuntimeFromCheckpointAndJournal(log, state)
	if err != nil {
		return err
	}
	r.SetEventLog(log)
	r.SetClock(brainruntime.NewFixedClock(now))
	b := teaching.Bridge{Runtime: r, Clock: brainruntime.NewFixedClock(now)}
	response, err := b.Exchange(raw)
	if err != nil {
		return err
	}
	p, _ := teaching.Unmarshal(raw)
	if p.Kind == teaching.Teach {
		if err = r.Checkpoint(state); err != nil {
			return fmt.Errorf("lesson may already be committed; recover journal before retry: %w", err)
		}
	}
	file, err := os.CreateTemp(filepath.Dir(*reply), "hnl-reply-*")
	if err != nil {
		return fmt.Errorf("request processed; could not create reply: %w", err)
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(response); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, *reply)
}
