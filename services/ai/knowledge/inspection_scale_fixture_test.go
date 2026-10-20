package knowledge

import (
	"math"
	"time"
)

// NewInspectionScaleFixture is available only in the knowledge test binary.
// It avoids quadratic input recruitment and excludes fixture loading buffers
// from a measurement of native runtime inspection/persistence operations.
func NewInspectionScaleFixture(count, fanout int) *Brain {
	b := NewBrain()
	b.Registry.byID = make(map[NodeID]*ConceptNode, count)
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= count; i++ {
		representation := make([]float64, 16)
		for j := range representation {
			representation[j] = math.Sin(float64(i*(j+3)) * .731)
		}
		n := &ConceptNode{ID: NodeID(i), Representation: representation, RestingActivation: .05, Threshold: .25, Frequency: 1, Importance: .5, Plasticity: .3, LastActivation: at}
		if fanout > 0 {
			n.Synapses = make(map[NodeID]SynapseList, fanout)
			for j := 1; j <= fanout; j++ {
				target := NodeID((i+j-1)%count + 1)
				n.Synapses[target] = SynapseList{&Synapse{TargetID: target, Weight: .2, Confidence: .5, Dynamic: DynamicState{Weight: .2, Confidence: .5, Frequency: 1}}}
			}
		}
		b.Registry.byID[n.ID] = n
	}
	b.Registry.nextID = NodeID(count + 1)
	return b
}
