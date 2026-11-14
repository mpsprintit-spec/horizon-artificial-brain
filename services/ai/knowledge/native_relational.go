package knowledge

import (
	"errors"
	"math"
	"slices"
	"sort"
)

const (
	NativeRelationalEncoderV1 = "local-relational-v1"
	NativeRelationalEncoderV2 = "local-relational-v2"
)

func (c NativeControlConfig) relationalFeatureCount() int {
	switch c.Encoder {
	case NativeRelationalEncoderV1:
		return 4
	case NativeRelationalEncoderV2:
		return 8
	default:
		return 0
	}
}

// ParameterSources are addresses in N, never a duplicate weight store.
func (s *NativeControlState) ParameterSources() []NodeID {
	ids := s.Config.Sources
	if s.Config.Encoder != "" {
		ids = s.Features
	}
	return append(append([]NodeID(nil), ids...), s.Bias)
}

func (s *NativeControlState) ReservedNodes() map[NodeID]bool {
	out := map[NodeID]bool{s.Bias: true}
	for _, id := range s.Features {
		out[id] = true
	}
	for _, group := range s.Outputs {
		for _, id := range group {
			out[id] = true
		}
	}
	return out
}

// Additive, permutation-invariant local graph moments. The four channels are
// activity and squared activity, each weighted by reciprocal and nonreciprocal
// connectivity. Fan-out normalizes population replication, without a graph
// scan, node-role labels, reward labels, depth, or a preferred operation.
// The projection from these channels to operations is learned in canonical N.
// V2 adds the signed counterparts of the same four moments. It preserves the
// V1 channels and traversal exactly, exposing inhibition already present in N.
// Neither version reads temporal eligibility or assigns semantic/context roles.
// This deliberately modest encoder cannot distinguish all relational graphs.
func nativeRelationalFeatures(s *NativeControlState, state map[NodeID]float64, get func(NodeID) *ConceptNode, counters ...*int) (map[NodeID]float64, error) {
	count := s.Config.relationalFeatureCount()
	if count == 0 || len(s.Features) != count || len(state) > s.Config.MaxActiveNodes {
		return nil, errors.New("relational feature dimensions/budget")
	}
	reserved := s.ReservedNodes()
	x := map[NodeID]float64{s.Bias: 1}
	reads := 0
	targets := make([]NodeID, 0, 8)
	for _, id := range nativeIDs(state) {
		v := state[id]
		if reserved[id] || v == 0 {
			continue
		}
		n := get(id)
		if n == nil {
			return nil, errors.New("relational source missing")
		}
		targets = targets[:0]
		for to := range n.Synapses {
			if !reserved[to] {
				targets = append(targets, to)
			}
		}
		// Stable accumulation for persistence/replay, independent of map iteration.
		slices.Sort(targets)
		mass, reciprocal, degree := 0.0, 0.0, 0.0
		signedMass, signedReciprocal := 0.0, 0.0
		for _, to := range targets {
			w := 0.0
			signed := 0.0
			var channelWeights []float64
			var signedWeights []float64
			if len(n.Synapses[to]) > 1 {
				channelWeights = make([]float64, 0, len(n.Synapses[to]))
				if count == 8 {
					signedWeights = make([]float64, 0, len(n.Synapses[to]))
				}
			}
			for _, edge := range n.Synapses[to] {
				reads++
				if reads > s.Config.MaxSynapseReads {
					return nil, errors.New("relational read budget exceeded")
				}
				if edge != nil {
					weight := math.Abs(edge.Weight * edge.Confidence)
					if count == 8 {
						value := weight
						if edge.Inhibitory {
							value = -value
						}
						if signedWeights != nil {
							signedWeights = append(signedWeights, value)
						} else {
							signed = value
						}
					}
					if channelWeights != nil {
						channelWeights = append(channelWeights, weight)
					} else {
						w = weight
					}
				}
			}
			if channelWeights != nil {
				sort.Float64s(channelWeights)
				for _, weight := range channelWeights {
					w += weight
				}
			}
			if signedWeights != nil {
				sort.Float64s(signedWeights)
				for _, value := range signedWeights {
					signed += value
				}
			}
			if w == 0 {
				continue
			}
			degree++
			mass += w
			signedMass += signed
			target := get(to)
			if target == nil {
				return nil, errors.New("relational target missing")
			}
			back := false
			for _, edge := range target.Synapses[id] {
				reads++
				if reads > s.Config.MaxSynapseReads {
					return nil, errors.New("relational read budget exceeded")
				}
				if edge != nil && edge.Weight*edge.Confidence != 0 {
					back = true
				}
			}
			if back {
				reciprocal += w
				signedReciprocal += signed
			}
		}
		r := 0.0
		if mass > 0 {
			r = reciprocal / mass
		}
		scale := 1 / math.Max(1, degree)
		channels := []float64{v * (1 - r), v * v * (1 - r), v * r, v * v * r}
		for j, z := range channels {
			x[s.Features[j]] += scale * z
		}
		if count == 8 && mass > 0 {
			a, b := (signedMass-signedReciprocal)/mass, signedReciprocal/mass
			for j, z := range []float64{v * a, v * v * a, v * b, v * v * b} {
				x[s.Features[4+j]] += scale * z
			}
		}
	}
	if len(counters) > 0 && counters[0] != nil {
		*counters[0] = reads
	}
	return x, nil
}
