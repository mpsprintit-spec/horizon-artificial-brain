package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func smNormalizeRevision(in GroundedDialogueReadout) GroundedDialogueReadout {
	in.Trace = append([]DialogueNumericTrace{}, in.Trace...)
	for i := range in.Trace {
		in.Trace[i].Readout.Sequence = 0
	}
	return in
}

func TestSelfModelAppendFailurePrecedesNeuralMutation(t *testing.T) {
	r := NewBrainRuntime(nil)
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	r.SetEventLog(log)
	before, _ := r.brain.CanonicalHash()
	log.beforeAppend = func(LoggedEvent) error { return errors.New("injected durable storage failure") }
	c := smConfig()
	if _, err = r.ApplyCognitiveModel(CognitiveModelCommand{ID: "fail-init", Initialize: &c}, time.Unix(1800000000, 0)); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("wrong append failure: %v", err)
	}
	after, _ := r.brain.CanonicalHash()
	if before != after || !r.RecoveryRequired() {
		t.Fatal("uncertain append changed neurons or left runtime usable")
	}
}

// This opt-in regression uses ONLY copies in Go's experiment temporary
// directory. No restore staging file is ever created alongside source memory.
func TestSelfModelRetainsTrainedV7Language(t *testing.T) {
	source := os.Getenv("HORIZON_SELFMODEL_RETENTION_SOURCE")
	casesPath := os.Getenv("HORIZON_SELFMODEL_RETENTION_CASES")
	if source == "" || casesPath == "" {
		t.Skip("set explicit internal V7 source/cases for archived language retention")
	}
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join(source, "language-brain.json"))
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(dir, "language-brain.json")
	if err = os.WriteFile(copyPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	r := NewBrainRuntime(nil)
	if err = r.RestoreCheckpoint(copyPath); err != nil {
		t.Fatal(err)
	}
	wal, e := os.ReadFile(filepath.Join(source, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	walPath := filepath.Join(dir, "events.jsonl")
	if e = os.WriteFile(walPath, wal, 0600); e != nil {
		t.Fatal(e)
	}
	journal, e := OpenEventLog(walPath)
	if e != nil {
		t.Fatal(e)
	}
	defer journal.Close()
	r.SetEventLog(journal)
	codecData, err := os.ReadFile(filepath.Join(source, "codec.json"))
	if err != nil {
		t.Fatal(err)
	}
	var codec DialogueCodec
	if err = json.Unmarshal(codecData, &codec); err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Request  GroundedDialogueRequest `json:"request"`
		Expected string                  `json:"expected"`
		Valid    bool                    `json:"valid"`
	}
	data, err = os.ReadFile(casesPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	before := make([]GroundedDialogueReadout, len(rows))
	for i, row := range rows {
		before[i], err = r.ReadGroundedDialogue(codec, row.Request)
		if err != nil {
			t.Fatal(err)
		}
	}
	canonicalBefore, _ := r.brain.CanonicalJSON()
	var graphBefore struct {
		Units      []json.RawMessage `json:"neural_units"`
		Synapses   []json.RawMessage `json:"synapses"`
		Resolution json.RawMessage   `json:"n_resolution"`
	}
	json.Unmarshal(canonicalBefore, &graphBefore)
	smInit(t, r)
	x := smInput()
	x.Reset = true
	for i := 0; i < 30; i++ {
		p, err := r.PreviewCognitiveModel(x)
		if err != nil {
			t.Fatal(err)
		}
		eps := []knowledge.CognitiveTrainingEpisode{{SuccessAt: 0, Frames: []knowledge.CognitiveTrainingFrame{{Features: x.Features, Mask: x.Mask, Action: p.Action, Draw: x.Draw, Behaviour: p.Policy, Reward: 1}}}}
		b := knowledge.CognitiveTrainingBatch{Episodes: eps, ParametersHash: p.ParametersHash, LearnPolicy: true, Evidence: knowledge.CognitiveEvidence{ID: "retention-outcome", Source: "isolated-test", Verifier: "controlled-success", Digest: knowledge.CognitiveDigest(eps)}}
		if _, err = r.ApplyCognitiveModel(CognitiveModelCommand{ID: "retain-" + time.Unix(int64(i), 0).Format(time.RFC3339), Training: &b}, time.Unix(1800000010+int64(i), 0)); err != nil {
			t.Fatal(err)
		}
	}
	canonicalAfter, _ := r.brain.CanonicalJSON()
	var graphAfter struct {
		Units      []json.RawMessage `json:"neural_units"`
		Synapses   []json.RawMessage `json:"synapses"`
		Resolution json.RawMessage   `json:"n_resolution"`
	}
	json.Unmarshal(canonicalAfter, &graphAfter)
	if !bytes.Equal(graphBefore.Resolution, graphAfter.Resolution) {
		t.Fatal("V7 representation/retention evidence changed")
	}
	oldIDs := map[knowledge.NodeID]bool{}
	for _, u := range graphBefore.Units {
		var id struct {
			ID knowledge.NodeID `json:"id"`
		}
		json.Unmarshal(u, &id)
		oldIDs[id.ID] = true
	}
	oldUnits, oldLinks := []json.RawMessage{}, []json.RawMessage{}
	for _, u := range graphAfter.Units {
		var id struct {
			ID knowledge.NodeID `json:"id"`
		}
		json.Unmarshal(u, &id)
		if oldIDs[id.ID] {
			oldUnits = append(oldUnits, u)
		}
	}
	for _, s := range graphAfter.Synapses {
		var id struct {
			Source knowledge.NodeID `json:"source_id"`
		}
		json.Unmarshal(s, &id)
		if oldIDs[id.Source] {
			oldLinks = append(oldLinks, s)
		}
	}
	if !reflect.DeepEqual(graphBefore.Units, oldUnits) || !reflect.DeepEqual(graphBefore.Synapses, oldLinks) {
		t.Fatal("cognitive-only training changed old neurons or synapses")
	}
	textSame, meaningSame := 0, 0
	for i, row := range rows {
		after, e := r.ReadGroundedDialogue(codec, row.Request)
		if e != nil {
			t.Fatal(e)
		}
		if after.Text == before[i].Text {
			textSame++
		}
		if after.MeaningValid == before[i].MeaningValid && after.Status == before[i].Status {
			meaningSame++
		}
		if !reflect.DeepEqual(smNormalizeRevision(before[i]), smNormalizeRevision(after)) {
			t.Fatalf("old response changed at archived case %d", i)
		}
	}
	path := filepath.Join(dir, "integrated.json")
	if err = r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	resumed := NewBrainRuntime(nil)
	if err = resumed.RestoreCheckpoint(path); err != nil {
		t.Fatal(err)
	}
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(journal, copyPath)
	if e != nil {
		t.Fatal(e)
	}
	left, _ := r.brain.CanonicalJSON()
	right, _ := replayed.brain.CanonicalJSON()
	if !bytes.Equal(left, right) {
		t.Fatal("trained V7 + self-model suffix replay differs")
	}
	for i, row := range rows {
		a, e := resumed.ReadGroundedDialogue(codec, row.Request)
		if e != nil || !reflect.DeepEqual(smNormalizeRevision(a), smNormalizeRevision(before[i])) {
			t.Fatalf("retention after restart case %d %v", i, e)
		}
	}
	result := map[string]any{"archived_cases": len(rows), "text_unchanged": textSame, "meaning_status_unchanged": meaningSame, "readout_fields_identical_except_runtime_revision": true, "old_units_unchanged": len(oldUnits), "old_synapses_unchanged": len(oldLinks), "n_resolution_unchanged": true, "restart_retention_equal": true, "scope": "archived regression, not fresh generalization"}
	result["v7_selfmodel_suffix_replay_equal"] = true
	result["integrated_sequence"] = r.LastSequence()
	if root := os.Getenv("HORIZON_SELFMODEL_WORKSPACE"); root != "" {
		dst := filepath.Join(root, "state", "v7-retention")
		if e = os.Mkdir(dst, 0700); e != nil {
			t.Fatal(e)
		}
		for name, src := range map[string]string{"initial.json": copyPath, "integrated.json": path, "codec.json": filepath.Join(source, "codec.json"), "events.jsonl": walPath, "retention-cases.json": casesPath} {
			b, e := os.ReadFile(src)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(dst, name), b, 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	t.Logf("retention %+v", result)
	if output := os.Getenv("HORIZON_SELFMODEL_RETENTION_REPORT"); output != "" {
		b, _ := json.MarshalIndent(result, "", "  ")
		if err = os.WriteFile(output, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
