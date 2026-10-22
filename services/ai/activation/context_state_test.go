package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestContextEligibilityIsBoundedByFrameAndMemberBudgets(t *testing.T) {
	state := &ContextState{}
	at := time.Unix(100, 0).UTC()
	frames := make([][]knowledge.NodeID, knowledge.ContextMaxFrames+20)
	for i := range frames {
		frames[i] = []knowledge.NodeID{knowledge.NodeID(i + 1)}
	}
	state.observe(&knowledge.ObservationTrace{Stimulus: frames}, at)
	if len(state.Frames) != knowledge.ContextMaxFrames {
		t.Fatal("frame bound not enforced")
	}
	large := make([]knowledge.NodeID, knowledge.ContextMaxMembers+1)
	for i := range large {
		large[i] = knowledge.NodeID(i + 1)
	}
	state.observe(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{large}}, at)
	if len(state.Frames) != knowledge.ContextMaxFrames {
		t.Fatal("oversized population was partially retained")
	}
	within := large[:knowledge.ContextMaxMembers]
	state.observe(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{within}}, at)
	if len(state.Frames) != 1 || len(state.Frames[0].Population) != knowledge.ContextMaxMembers {
		t.Fatal("member bound did not evict complete older populations")
	}
	if err := state.Validate(func(knowledge.NodeID) bool { return true }); err != nil {
		t.Fatal(err)
	}
	if len(state.before(at.Add(knowledge.ContextMaxAge+time.Nanosecond), false)) != 0 {
		t.Fatal("expired context remains eligible")
	}
}
