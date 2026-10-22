package activation

import (
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// A valid substrate has at most one channel per target/polarity/Kind. Kind is
// only a compatibility tie-breaker for old imported channels, never a semantic
// instruction. This sorts a detached pointer slice, not the stored connections.
func orderPropagationSynapses(synapses []*knowledge.Synapse, canonical bool) {
	sort.Slice(synapses, func(i, j int) bool {
		a, b := synapses[i], synapses[j]
		if canonical && a.TargetID == b.TargetID {
			if a.Inhibitory != b.Inhibitory {
				return !a.Inhibitory
			}
			return a.Kind < b.Kind
		}
		return a.TargetID < b.TargetID
	})
}
