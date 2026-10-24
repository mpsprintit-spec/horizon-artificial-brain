package main

import (
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"os"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// initializeProductionHorizon establishes exactly one serving brain under its
// supervisor BEFORE any HTTP or continuous service starts. The committed
// journal and authenticated checkpoint are authoritative; never graft an old
// event index onto an arbitrary memory-file state.
//
// New deployments receive a sequence-zero anchored checkpoint before their
// first input. Legacy memory-only deployments are allowed only when their
// journal is empty, and their imported state is anchored before accepting new
// writes. A legacy memory file plus a nonempty journal without a checkpoint
// must match full genesis replay; otherwise startup fails closed.
func initializeProductionHorizon(log *runtime.EventLog, journalPath, memoryPath, checkpointPath string) (*core.HorizonEngine, bool, error) {
	if log == nil || journalPath == "" || memoryPath == "" || checkpointPath == "" {
		return nil, false, errors.New("startup requires an open journal and explicit paths")
	}
	events, err := runtime.ReadEventLog(journalPath)
	if err != nil { return nil, false, fmt.Errorf("verify startup committed journal: %w",err) }

	checkpointExists, err := startupFileExists(checkpointPath)
	if err != nil { return nil, false, fmt.Errorf("stat checkpoint: %w",err) }
	memoryExists, err := startupFileExists(memoryPath)
	if err != nil { return nil, false, fmt.Errorf("stat brain memory: %w",err) }
	h := core.NewHorizonEngine()
	if h == nil || h.Supervisor() == nil { return nil, false, errors.New("cannot initialize canonical Horizon engine") }

	if checkpointExists {
		if err := h.Supervisor().RecoverFromCheckpoint(log, checkpointPath); err != nil {
			return nil, false, fmt.Errorf("verified checkpoint recovery failed; refusing unsafe bootstrap: %w",err)
		}
		// If the process stopped after its sequence-zero checkpoint but
		// before the very first bootstrap event, complete bootstrap on
		// restart instead of silently serving an uninitialized substrate.
		return h, len(events) == 0 && !memoryExists, nil
	}

	// Enabled serving memory is a complete authenticated checkpoint, not a
	// graph-only export. If the primary checkpoint is missing, verify its
	// journal anchor instead of trying to import its envelope as a bare Brain.
	if memoryExists {
		data,readErr:=os.ReadFile(memoryPath);if readErr!=nil{return nil,false,readErr}
		var header struct{BrainIdentity string `json:"brain_identity"`;Brain json.RawMessage `json:"brain"`}
		if json.Unmarshal(data,&header)==nil&&header.BrainIdentity==runtime.BrainIdentity&&len(header.Brain)>0 {
			if err:=h.Supervisor().RecoverFromCheckpoint(log,memoryPath);err!=nil{return nil,false,fmt.Errorf("complete serving memory recovery refused: %w",err)}
			if err:=h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime)error{return rt.Checkpoint(checkpointPath)});err!=nil{return nil,false,err}
			return h,false,nil
		}
	}

	if len(events) > 0 {
		if err := h.Supervisor().RecoverFromCommittedJournal(log, knowledge.NewBootstrapBrain); err != nil {
			return nil, false, fmt.Errorf("committed genesis replay failed; refusing unsafe bootstrap: %w",err)
		}
		// Historical deployments sometimes started from an unjournaled
		// memory file and then began appending events. Refuse silent memory
		// loss if that file disagrees with reconstructible state.
		if memoryExists {
			diskBrain := knowledge.NewBrain()
			if err := diskBrain.Load(memoryPath); err != nil {
				return nil, false, fmt.Errorf("cannot validate existing brain file against journal: %w", err)
			}
			disk, err := diskBrain.CanonicalJSON()
			if err != nil { return nil, false, err }
			inspection, err := h.InspectActiveBrain()
			if err != nil { return nil, false, err }
			recovered := inspection.Graph.CanonicalBytes()
			if !bytes.Equal(disk, recovered) {
				return nil, false, errors.New("legacy brain memory disagrees with committed journal; checkpoint or explicit migration required")
			}
		}
		// A v7 checkpoint now anchors any future recovery to this exact
		// committed prefix, eliminating dependence on a raw memory alias.
		if err := h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
			return rt.Checkpoint(checkpointPath)
		}); err != nil { return nil, false, fmt.Errorf("anchor replayed runtime: %w",err) }
		return h, false, nil
	}

	// The journal is empty and there is no checkpoint. Importing a historic
	// standalone Brain is safe only before connecting the journal, and the
	// imported state is immediately captured by the initial anchored v7
	// checkpoint. It is never merged with a nonempty committed history.
	if memoryExists {
		if err := h.LoadActiveBrain(memoryPath); err != nil { return nil, false, fmt.Errorf("load standalone brain: %w",err) }
	}
	if err := h.Supervisor().AttachEventLog(log); err != nil { return nil, false, fmt.Errorf("attach genesis journal: %w", err) }
	if err := h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
		return rt.Checkpoint(checkpointPath)
	}); err != nil { return nil, false, fmt.Errorf("create genesis checkpoint: %w",err) }
	return h, !memoryExists, nil
}

func startupFileExists(path string) (bool,error) {
	_,err:=os.Stat(path)
	if err == nil { return true,nil }
	if errors.Is(err,os.ErrNotExist) { return false,nil }
	return false,err
}
