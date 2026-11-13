package runtime

import (
	"bytes"
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
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
