package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func terminalUtilityRuntimeBatch(t *testing.T, rt *BrainRuntime, objective string) k.CoordinatedBatchV5 {
	t.Helper()
	old := coordinatedRuntimeBatch(t, rt)
	c := k.TerminalUtilityCaseV5{Trace: old.Episodes[0].Traces[0]}
	c.Answers = [4]k.TerminalAnswerCountsV5{{Observed: 4, Correct: 2}, {Observed: 4, Correct: 1}, {Observed: 4, Correct: 0}, {Observed: 4, Correct: 1}}
	p := &k.TerminalUtilityBatchV5{Protocol: k.TerminalUtilityProtocolV5, Objective: objective, DatasetSHA256: strings.Repeat("a", 64), Cases: []k.TerminalUtilityCaseV5{c}}
	return k.CoordinatedBatchV5{TerminalUtility: p, ParametersHash: old.ParametersHash, Evidence: k.CognitiveEvidence{
		ID: "observed-terminal-counts", Source: "unit-selected-outcome-fixture", Verifier: "uniform answer sufficient statistics", Digest: k.CognitiveDigest(p)}}
}
func TestTerminalUtilityV5NativeOwnershipReplayAndGuards(t *testing.T) {
	for _, objective := range []string{"policy_value", "logscore"} {
		t.Run(objective, func(t *testing.T) {
			rt, wal, base := phase4Fixture(t)
			spec := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
			at := time.Unix(2800000000, 0)
			if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-alignment-head", Phase4Init: &spec}, at); e != nil {
				t.Fatal(e)
			}
			w0, e := rt.CognitiveModelParameters()
			if e != nil {
				t.Fatal(e)
			}
			b := terminalUtilityRuntimeBatch(t, rt, objective)
			cmd := CognitiveModelCommand{ID: "terminal-alignment-update", Coordinated: &b}
			out, e := rt.ApplyCognitiveModel(cmd, at.Add(time.Second))
			if e != nil {
				t.Fatal(e)
			}
			if out.Coordinated == nil || out.Coordinated.UpdatedHead != 1097 || out.Coordinated.UpdatedMUSE != 0 {
				t.Fatal("wrong update ownership")
			}
			w1, e := rt.CognitiveModelParameters()
			if e != nil {
				t.Fatal(e)
			}
			start := len(w1) - 1097
			if k.CognitiveDigest(w0[:start]) != k.CognitiveDigest(w1[:start]) || k.CognitiveDigest(w0) == k.CognitiveDigest(w1) {
				t.Fatal("frozen prefix moved or no update")
			}
			dup, e := rt.ApplyCognitiveModel(cmd, at.Add(time.Second))
			if e != nil || !dup.Duplicate {
				t.Fatal("retry", e)
			}
			path := filepath.Join(t.TempDir(), "snapshot.json")
			if e = rt.Checkpoint(path); e != nil {
				t.Fatal(e)
			}
			var cp BrainSnapshot
			raw, _ := os.ReadFile(path)
			if e = json.Unmarshal(raw, &cp); e != nil || cp.SchemaVersion != 19 {
				t.Fatal("schema", e)
			}
			events, e := ReadEventLog(wal.file.Name())
			if e != nil {
				t.Fatal(e)
			}
			if events[len(events)-1].SchemaVersion != 13 {
				t.Fatal("wrong WAL schema")
			}
			replay, e := RebuildRuntimeFromCheckpointAndJournal(wal, base)
			if e != nil {
				t.Fatal(e)
			}
			want, _ := rt.brain.CanonicalJSON()
			got, _ := replay.brain.CanonicalJSON()
			if !bytes.Equal(want, got) {
				t.Fatal("replay changed canonical state")
			}
			for _, fault := range []string{"digest", "objective", "count", "mask", "mixed", "dataset", "budget", "reset"} {
				bad := terminalUtilityRuntimeBatch(t, rt, objective)
				switch fault {
				case "digest":
					bad.Evidence.Digest = "invalid"
				case "objective":
					bad.TerminalUtility.Objective = "arbitrary"
				case "count":
					bad.TerminalUtility.Cases[0].Answers[0].Correct = 99
				case "mask":
					bad.TerminalUtility.Cases[0].Trace[0].Mask[3] = false
				case "mixed":
					bad.Episodes = coordinatedRuntimeBatch(t, rt).Episodes
				case "dataset":
					bad.TerminalUtility.DatasetSHA256 = strings.Repeat("z", 64)
				case "budget":
					bad.TerminalUtility.Cases[0].Trace[0].NeuralMeta.Remaining = 2
				case "reset":
					bad.TerminalUtility.Cases[0].Trace[0].Reset = false
				}
				if fault != "digest" {
					bad.Evidence.Digest = k.CognitiveDigest(bad.TerminalUtility)
				}
				before, _ := rt.brain.CanonicalHash()
				_, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "reject-" + fault, Coordinated: &bad}, at)
				after, _ := rt.brain.CanonicalHash()
				if e == nil || before != after || rt.RecoveryRequired() {
					t.Fatalf("invalid %s mutated or passed: %v", fault, e)
				}
			}
		})
	}
}
