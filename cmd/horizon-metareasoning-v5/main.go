// Opt-in native V5 workbench. It never participates in ordinary device/chat
// dispatch, and never chooses a default state directory or imports seed101.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"io"
	"os"
	"path/filepath"
	"time"
)

func decode(path string, out any) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 16<<20))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		return e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("exactly one JSON request required")
	}
	return nil
}
func runArgs(args []string) error {
	flags := flag.NewFlagSet("horizon-metareasoning-v5", flag.ContinueOnError)
	enabled := flags.Bool("experimental-v5", false, "explicitly enable experimental V5 API (default off)")
	state := flags.String("state", "", "explicit isolated state directory")
	request := flags.String("request", "", "observed-input trace JSON, read-only")
	command := flags.String("command", "", "versioned runtime command JSON, journaled")
	if e := flags.Parse(args); e != nil {
		return e
	}
	if !*enabled || *state == "" || (*request == "") == (*command == "") {
		return errors.New("requires -experimental-v5, explicit -state, and exactly one of -request or -command")
	}
	cp := filepath.Join(*state, "brain.json")
	walPath := filepath.Join(*state, "events.jsonl")
	_, statErr := os.Stat(cp)
	exists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if exists {
		if _, e := os.Stat(walPath); e != nil {
			return fmt.Errorf("checkpoint requires its authentic journal: %w", e)
		}
	}
	if e := os.MkdirAll(*state, 0700); e != nil {
		return e
	}
	wal, e := r.OpenEventLog(walPath)
	if e != nil {
		return e
	}
	defer wal.Close()
	rt := r.NewBrainRuntime(k.NewBrain())
	if exists {
		rt, e = r.RebuildRuntimeFromCheckpointAndJournal(wal, cp)
		if e != nil {
			return e
		}
	} else {
		events, err := r.ReadEventLog(walPath)
		if err != nil {
			return err
		}
		if len(events) > 0 {
			if err = r.ReplayEventLog(rt, events); err != nil {
				return err
			}
		} else if *request != "" {
			return errors.New("preview requires an initialized profile")
		}
	}
	rt.SetEventLog(wal)
	if *request != "" {
		var trace []k.CognitiveInput
		if e = decode(*request, &trace); e != nil {
			return e
		}
		out, err := rt.PreviewCoordinatedV5(trace, k.Phase4DropV5{})
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	var c r.CognitiveModelCommand
	if e = decode(*command, &c); e != nil {
		return e
	}
	out, e := rt.ApplyCognitiveModel(c, time.Now().UTC())
	if e != nil {
		return e
	}
	if e = rt.Checkpoint(cp); e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
func main() {
	if e := runArgs(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
