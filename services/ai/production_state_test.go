package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func canonicalStartup(t *testing.T, paths productionPaths) (*productionState, *core.HorizonEngine) {
	t.Helper()
	state, err := openProductionState(paths)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(state.Close)
	h, bootstrap, err := initializeProductionHorizon(state.Log, paths.Journal, paths.Memory, paths.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Supervisor().Close() })
	if bootstrap {
		t.Fatal("recovered canonical state requested empty bootstrap")
	}
	output, stimuli, err := finishProductionStartup(h, bootstrap, paths.Memory, paths.Checkpoint)
	if err != nil || stimuli != 0 {
		t.Fatalf("startup mutated restored Brain: stimuli=%d err=%v", stimuli, err)
	}
	inspection, err := h.InspectActiveBrain()
	if err != nil || output.Sequence != inspection.Revision {
		t.Fatal("startup did not publish recovered authority", err)
	}
	return state, h
}

func startupNativeState(t *testing.T, h *core.HorizonEngine) *knowledge.NativeControlState {
	t.Helper()
	var brain struct {
		Native *knowledge.NativeControlState `json:"native_control"`
	}
	if err := json.Unmarshal(getActiveStartupState(t, h), &brain); err != nil || brain.Native == nil {
		t.Fatal("native learned state missing", err)
	}
	return brain.Native
}

func TestCanonicalProductionStartupContinuesAndRestarts(t *testing.T) {
	cwd := t.TempDir()
	paths, err := resolveProductionPaths(func(string) string { return "" }, cwd)
	if err != nil || paths.Directory != filepath.Join(cwd, "horizon-state") {
		t.Fatal("fresh installation not canonical", err)
	}
	state, h := canonicalStartup(t, paths)
	initial := getActiveStartupState(t, h)
	inspection, err := h.InspectActiveBrain()
	if err != nil {
		t.Fatal(err)
	}
	u, s, p := inspection.Graph.Counts()
	if inspection.Revision != 75331 || u != 57 || s != 94 || p != 24 || startupNativeState(t, h).Updates != 10752 {
		t.Fatal("wrong canonical initial state")
	}
	// RuntimeSupervisor and the existing journal lock both fence a second
	// production writer. No adapter holds a second active Brain pointer.
	if other, err := openProductionState(paths); err == nil {
		other.Close()
		t.Fatal("two serving owners accepted")
	}
	if other, err := runtime.OpenEventLog(paths.Journal); err == nil {
		other.Close()
		t.Fatal("journal owner bypassed directory lock")
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	state.Close()
	state, h = canonicalStartup(t, paths)
	if !bytes.Equal(initial, getActiveStartupState(t, h)) {
		t.Fatal("idle restart changed learned state")
	}
	// One new on-policy experience tests the existing learning/WAL path. It
	// is not acquisition training or an A–E1 experiment, and only this
	// disposable production directory receives the experience.
	native := startupNativeState(t, h)
	at := time.Date(2026, 11, 14, 0, 1, 16, 0, time.UTC)
	var last runtime.NativeControlResult
	err = h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
		apply := func(c runtime.NativeControlCommand) error {
			at = at.Add(time.Millisecond)
			var e error
			last, e = rt.ApplyNativeControl(c, at)
			return e
		}
		if e := apply(runtime.NativeControlCommand{ID: "production-begin", Begin: &runtime.NativeBegin{EpisodeID: "production-experience", Training: true, Observation: &runtime.NativeObservation{Source: "production-regression", SampleID: "new-observation", State: map[knowledge.NodeID]float64{native.Config.Sources[0]: .8}}}}); e != nil {
			return e
		}
		for step := 0; step < native.Config.MaxSteps; step++ {
			if e := apply(runtime.NativeControlCommand{ID: fmt.Sprintf("production-step-%d", step), Step: &runtime.NativeStep{EpisodeID: "production-experience", Sample: .01}}); e != nil {
				return e
			}
			if last.Stopped || last.Truncated {
				break
			}
		}
		return apply(runtime.NativeControlCommand{ID: "production-outcome", Outcome: &runtime.NativeOutcome{EpisodeID: "production-experience", TraceDigest: last.TraceDigest, Source: "production-regression", OutcomeID: "observed-outcome", Provenance: "observed-environment", Reward: .7, Learn: true}})
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Sequence <= 75331 || last.Learning == nil || startupNativeState(t, h).Updates != 10753 {
		t.Fatalf("learning did not continue A–E1: %+v", last)
	}
	continued := getActiveStartupState(t, h)
	if bytes.Equal(initial, continued) {
		t.Fatal("observed learning did not update canonical substrate")
	}
	// Crash boundary: leave the original checkpoint untouched. Only WAL
	// suffix recovery can reconstruct the new experience on this restart.
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	state.Close()
	state, h = canonicalStartup(t, paths)
	if !bytes.Equal(continued, getActiveStartupState(t, h)) || startupNativeState(t, h).Updates != 10753 {
		t.Fatal("WAL suffix lost continued learning")
	}
	if err := h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
		if rt.LastSequence() != last.Sequence {
			t.Fatal("restart reset journal sequence")
		}
		if err := rt.SaveBrain(paths.Memory); err != nil {
			return err
		}
		return rt.Checkpoint(paths.Checkpoint)
	}); err != nil {
		t.Fatal(err)
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	state.Close()
	_, h = canonicalStartup(t, paths)
	if !bytes.Equal(continued, getActiveStartupState(t, h)) {
		t.Fatal("checkpoint restart lost learned state")
	}
	t.Logf("A–E1 production: sequence 75331 -> %d, native updates 10752 -> 10753; idle, WAL and checkpoint restarts passed", last.Sequence)
}

func TestCanonicalProductionPreservesLegacyAndSupportsRollback(t *testing.T) {
	cwd := t.TempDir()
	memory := filepath.Join(cwd, "brain_memory.json")
	old := knowledge.NewBrain()
	old.Store("existing-user-experience")
	if err := old.Save(memory); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := resolveProductionPaths(func(string) string { return "" }, cwd)
	if err != nil || legacy.Directory != "" || legacy.Memory != memory {
		t.Fatal("existing state not prioritized", err)
	}
	state, h := canonicalStartup(t, legacy)
	before := getActiveStartupState(t, h)
	if !bytes.Equal(original, before) {
		t.Fatal("legacy experience replaced")
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	state.Close()
	// Back up the stopped legacy deployment as one matching pair, then
	// explicitly select a separate A–E1 directory. Rollback selects the
	// preserved original paths, never merges the two journals.
	backup := filepath.Join(cwd, "backup")
	if err := os.Mkdir(backup, 0700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{legacy.Memory, legacy.Checkpoint, legacy.Journal} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(backup, filepath.Base(p)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	activated := filepath.Join(cwd, "ae1-explicit")
	if err := prepareCanonicalAE1(activated); err != nil {
		t.Fatal(err)
	}
	if err := prepareCanonicalAE1(activated); err == nil {
		t.Fatal("explicit preparation overwrote existing state")
	}
	selected, err := resolveProductionPaths(func(key string) string {
		if key == "HORIZON_STATE_DIR" {
			return activated
		}
		return ""
	}, cwd)
	if err != nil {
		t.Fatal(err)
	}
	state, h = canonicalStartup(t, selected)
	if startupNativeState(t, h).Updates != 10752 {
		t.Fatal("activation did not load canonical learned Brain")
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	state.Close()
	for _, p := range []string{legacy.Memory, legacy.Checkpoint, legacy.Journal} {
		want, err := os.ReadFile(filepath.Join(backup, filepath.Base(p)))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("activation mutated old lineage", p, err)
		}
	}
	_, h = canonicalStartup(t, legacy)
	if !bytes.Equal(before, getActiveStartupState(t, h)) {
		t.Fatal("rollback changed pre-activation experience")
	}
}

func TestCanonicalProductionRefusesInvalidActivationWithoutPublication(t *testing.T) {
	original := canonicalAE1Bundle
	canonicalAE1Bundle = []byte("invalid compressed state")
	defer func() { canonicalAE1Bundle = original }()
	destination := filepath.Join(t.TempDir(), "new-state")
	if err := prepareCanonicalAE1(destination); err == nil {
		t.Fatal("invalid source activated")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("failed validation published a directory")
	}
}

func TestCanonicalProductionRejectsMismatchedPairWithoutMutation(t *testing.T) {
	paths := directoryProductionPaths(filepath.Join(t.TempDir(), "state"))
	if err := prepareCanonicalAE1(paths.Directory); err != nil {
		t.Fatal(err)
	}
	// Replace the checkpoint by another authenticated lineage's genesis.
	other := t.TempDir()
	cp := filepath.Join(other, "checkpoint.json")
	wal := newStartupJournal(t, filepath.Join(other, "events.jsonl"))
	h, _, err := initializeProductionHorizon(wal, filepath.Join(other, "events.jsonl"), filepath.Join(other, "memory.json"), cp)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Supervisor().Close()
	foreign, err := os.ReadFile(cp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Checkpoint, foreign, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(paths.Journal)
	if err != nil {
		t.Fatal(err)
	}
	if state, err := openProductionState(paths); err == nil {
		state.Close()
		t.Fatal("foreign checkpoint accepted")
	}
	after, err := os.ReadFile(paths.Journal)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected pairing mutated journal", err)
	}
	if got, err := os.ReadFile(paths.Checkpoint); err != nil || !bytes.Equal(got, foreign) {
		t.Fatal("rejected pairing mutated checkpoint", err)
	}
}

func TestCanonicalProductionConfigurationFailsClosed(t *testing.T) {
	cwd := t.TempDir()
	for _, env := range []map[string]string{
		{"HORIZON_STATE_DIR": "state", "HORIZON_EVENT_LOG_PATH": "foreign.jsonl"},
		{"HORIZON_BRAIN_MEMORY_PATH": "missing-brain.json"},
	} {
		if _, err := resolveProductionPaths(func(k string) string { return env[k] }, cwd); err == nil {
			t.Fatal("ambiguous or empty legacy configuration accepted")
		}
	}
	dir := filepath.Join(cwd, "unrelated")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := openProductionState(directoryProductionPaths(dir)); err == nil {
		s.Close()
		t.Fatal("unrelated directory replaced")
	}
	link := filepath.Join(cwd, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if s, err := openProductionState(directoryProductionPaths(link)); err == nil {
		s.Close()
		t.Fatal("state-directory symlink accepted")
	}
}

func TestCanonicalProductionRejectsInvalidMemoryBeforeJournalCreation(t *testing.T) {
	for name, data := range map[string][]byte{
		"malformed":   []byte("{invalid existing state}\n"),
		"checkpoint":  []byte(`{"schema_version":20,"brain_identity":"horizon-primary-brain","brain":{}}`),
		"unsupported": []byte(`{"schema_version":7,"native_control":{}}`),
	} {
		t.Run(name, func(t *testing.T) {
			paths := directoryProductionPaths(t.TempDir())
			if err := os.WriteFile(paths.Memory, data, 0600); err != nil {
				t.Fatal(err)
			}
			if state, err := openProductionState(paths); err == nil {
				state.Close()
				t.Fatal("invalid memory accepted before journal creation")
			}
			got, err := os.ReadFile(paths.Memory)
			if err != nil || !bytes.Equal(got, data) {
				t.Fatal("rejected startup changed existing memory", err)
			}
			entries, err := os.ReadDir(paths.Directory)
			if err != nil || len(entries) != 1 || entries[0].Name() != "brain_memory.json" {
				t.Fatal("rejected startup created state files", entries, err)
			}
		})
	}
}
