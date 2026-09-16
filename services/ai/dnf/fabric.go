package dnf

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Fabric is the DNF boundary over Horizon's one canonical Brain substrate.
// It deliberately owns no neural storage. Nodes, synapses, projections and
// persistence remain in knowledge.Brain; this type only provides structural
// inspection and substrate-safe operations.
type Fabric struct {
	brain *knowledge.Brain
}

func NewFabric(brain *knowledge.Brain) (*Fabric, error) {
	if brain == nil {
		return nil, errors.New("dnf requires a brain substrate")
	}
	return &Fabric{brain: brain}, nil
}

// Brain returns the canonical substrate reference. No copy or secondary store
// is created.
func (f *Fabric) Brain() *knowledge.Brain {
	if f == nil {
		return nil
	}
	return f.brain
}

// Structure describes the current DNF substrate without duplicating its state.
type Structure struct {
	Nodes              int
	DynamicSynapses    int
	LegacySynapses     int
	ProjectionFields   int
	ObservedAt         time.Time
}

// Inspect derives structural metrics directly from the canonical Brain.
func (f *Fabric) Inspect(now time.Time) (Structure, error) {
	if f == nil || f.brain == nil || f.brain.Registry == nil {
		return Structure{}, errors.New("dnf is not initialized")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	structure := Structure{ObservedAt: now.UTC()}
	nodes := f.brain.Registry.Nodes()
	structure.Nodes = len(nodes)
	for _, node := range nodes {
		if node == nil {
			continue
		}
		for _, synapses := range node.Synapses {
			for _, synapse := range synapses {
				if synapse != nil && synapse.IsDynamic() {
					structure.DynamicSynapses++
				} else if synapse != nil {
					structure.LegacySynapses++
				}
			}
		}
	}
	structure.ProjectionFields = len(f.brain.ProjectionPopulations)
	return structure, nil
}

// Ground projects an observation into the same canonical brain used by every
// other Horizon subsystem. DNF never creates an alternate representation store.
func (f *Fabric) Ground(vector knowledge.NeuralVector) (knowledge.GroundedVector, error) {
	if f == nil || f.brain == nil {
		return knowledge.GroundedVector{}, errors.New("dnf is not initialized")
	}
	return f.brain.GroundVector(vector)
}
