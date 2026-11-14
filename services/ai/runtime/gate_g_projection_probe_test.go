package runtime

import (
	"bytes"
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"testing"
)

// An observation equal to a policy prototype must create/reuse WORLD units,
// including through the historical unscoped numeric projector.
func TestGateGAutomaticProjectionCannotRecruitNativePolicy(t *testing.T) {
	for _, dynamics := range []k.DynamicsEncoding{k.LegacyDynamics, k.CurrentDynamics} {
		name := string(dynamics)
		if name == "" {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			rt, ids := nativeFixture(t)
			rt.dynamicsEncoding = dynamics
			wal := openKnowledgeJournal(t)
			rt.SetEventLog(wal)
			cfg := nativeControlConfig(ids)
			nativeApply(t, rt, NativeControlCommand{ID: "init-projection-probe", Initialize: &cfg})
			s := rt.brain.NativeControlSnapshot()
			reserved := s.ReservedNodes()
			snapshot := func() []byte {
				nodes := []*k.ConceptNode{}
				for _, id := range []k.NodeID{s.Bias} {
					nodes = append(nodes, rt.brain.Registry.GetByID(id))
				}
				b, e := json.Marshal(nodes)
				if e != nil {
					t.Fatal(e)
				}
				return b
			}
			before := snapshot()
			vector := k.NewNeuralVector(rt.brain.Registry.GetByID(s.Bias).Representation)
			grounded, e := rt.GroundVectorAt(vector, gateGTime(rt))
			if e != nil {
				t.Fatal(e)
			}
			for _, id := range grounded.Population {
				if reserved[id] {
					t.Fatalf("automatic world projection recruited native policy unit %d", id)
				}
			}
			if !bytes.Equal(before, snapshot()) {
				t.Fatal("world projection mutated native bias state")
			}
			cp := filepath.Join(t.TempDir(), "projected.json")
			if e = rt.Checkpoint(cp); e != nil {
				t.Fatal(e)
			}
			restored, e := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
			if e != nil {
				t.Fatal(e)
			}
			gateGCompare(t, rt, restored)
		})
	}
}

// A valid checksum is not permission for world provenance to alias policy cells.
// Reject the old unsafe state without rewriting the supplied checkpoint.
func TestGateGRecoveryRejectsPolicyWorldProjectionAlias(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "alias-init", Initialize: &cfg})
	s := rt.brain.NativeControlSnapshot()
	data := mustGateGBrain(t, rt)
	var raw map[string]json.RawMessage
	if e := json.Unmarshal(data, &raw); e != nil {
		t.Fatal(e)
	}
	var populations []k.ProjectionPopulation
	if p := raw["populations"]; len(p) > 0 {
		if e := json.Unmarshal(p, &populations); e != nil {
			t.Fatal(e)
		}
	}
	populations = append(populations, k.ProjectionPopulation{
		Prototype: append([]float64(nil), rt.brain.Registry.GetByID(s.Bias).Representation...),
		Units:     []k.PopulationUnit{{NodeID: s.Bias, Activation: 1}},
	})
	raw["populations"], _ = json.Marshal(populations)
	unsafe, e := json.Marshal(raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = k.ValidateNativeControlState(unsafe, true); e == nil {
		t.Error("unsafe world-policy alias accepted by native validator")
	}
	cp := filepath.Join(t.TempDir(), "valid.json")
	if e = rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(cp)
	if e != nil {
		t.Fatal(e)
	}
	var snapshot BrainSnapshot
	if e = json.Unmarshal(b, &snapshot); e != nil {
		t.Fatal(e)
	}
	snapshot.Brain = unsafe
	snapshot.Checksum, e = checkpointChecksumForSchema(snapshot)
	if e != nil {
		t.Fatal(e)
	}
	b, e = json.Marshal(snapshot)
	if e != nil {
		t.Fatal(e)
	}
	bad := filepath.Join(t.TempDir(), "unsafe-valid-checksum.json")
	if e = os.WriteFile(bad, b, 0600); e != nil {
		t.Fatal(e)
	}
	fresh := NewBrainRuntime(k.NewBrain())
	before := mustGateGBrain(t, fresh)
	if e = fresh.RestoreCheckpoint(bad); e == nil {
		t.Error("unsafe checkpoint restored despite world-policy alias")
	}
	if !bytes.Equal(before, mustGateGBrain(t, fresh)) {
		t.Error("failed restore mutated target Brain")
	}
	after, e := os.ReadFile(bad)
	if e != nil || !bytes.Equal(b, after) {
		t.Fatal("recovery rewrote unsafe checkpoint", e)
	}
}
