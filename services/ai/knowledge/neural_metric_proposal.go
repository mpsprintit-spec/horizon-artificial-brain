package knowledge

// Proposal workspaces are disposable. Only an admitted model is written to
// the existing signed synapses by learnNMetric. This is not a second learner.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"slices"
)

const NMetricBoundedV1 = "incumbent-relative-v1"
const NMetricRefineV1 = "incumbent-relative-leaf-refinement-v1"
const NMetricBlockV1 = "incumbent-relative-block-refinement-v1"
const NMetricAuditV1 = "incumbent-relative-audited-v1"

// Reuse is a generic opt-in lesson on the existing basis. New knowledge is
// written by the ordinary signed head writer; no separate estimator is kept.
const NMetricReuseV1 = "native-shared-basis-reuse-v1"
const NMetricReuseV2 = "native-shared-basis-reuse-v2"
const NMetricReuseV3 = "native-shared-basis-regularized-v3"

func nMetricReuse(p string) bool {
	return p == NMetricReuseV1 || p == NMetricReuseV2 || p == NMetricReuseV3
}

type NMetricProposalConfig struct {
	CoordinateBlocks [][]int   `json:"coordinate_blocks,omitempty"`
	Protocol         string    `json:"protocol"`
	Strategy         string    `json:"strategy"` // full-refit, new-feature, bounded-joint, combined
	Fractions        []float64 `json:"fractions,omitempty"`
	PreserveShared   bool      `json:"preserve_shared,omitempty"`
	// Explicit fixed-capacity research intervention. Zero preserves the
	// historical 32-feature law; this is not an adaptive growth policy.
	FeatureLimit int `json:"feature_limit,omitempty"`
	// Explicit research search within the existing feature budget. Only
	// task-local leaf directions may change; every existing gate still runs.
	RefineLeafDirections bool    `json:"refine_leaf_directions,omitempty"`
	Update               string  `json:"update,omitempty"`
	BlockSparse          bool    `json:"block_sparse,omitempty"`
	FixedBasis           bool    `json:"fixed_basis,omitempty"`
	Ridge                float64 `json:"ridge,omitempty"`
}

func nMetricFeatureLimit(p *NMetricProposalConfig) int {
	if p != nil && p.FeatureLimit != 0 {
		return p.FeatureLimit
	}
	return maxNFeatures
}

func validNFeatureLimit(limit int) bool { return limit == 0 || limit == 32 || limit == 64 }

// Evidence hashes include the actual encoded measurements, labels, uncertainty,
// query and episode IDs. Different field names do not establish independence.
type NMetricEvidence struct {
	Fit, Check, WitnessFit, WitnessExcluded int
	FitSHA256, CheckSHA256, WitnessSHA256   string
}

func validateNMetricProposal(p *NMetricProposalConfig) error {
	if p == nil {
		return nil
	}
	if !nFinite(p.Ridge) || (p.Protocol == NMetricReuseV3 && (p.Ridge <= 0 || p.Ridge > 1)) || (p.Protocol != NMetricReuseV3 && p.Ridge != 0) {
		return errors.New("regularization requires explicit versioned fixed-basis protocol")
	}
	if !validNFeatureLimit(p.FeatureLimit) {
		return errors.New("experimental numerical feature limit must be 32 or 64")
	}
	if p.Protocol != NMetricBoundedV1 && p.Protocol != NMetricRefineV1 && p.Protocol != NMetricAuditV1 && p.Protocol != NMetricConstrainedV1 && p.Protocol != NMetricBlockV1 && !nMetricReuse(p.Protocol) {
		return errors.New("unknown numerical proposal protocol")
	}
	if (p.RefineLeafDirections && p.Protocol != NMetricRefineV1 && p.Protocol != NMetricConstrainedV1 && p.Protocol != NMetricBlockV1) || ((p.Protocol == NMetricRefineV1 || p.Protocol == NMetricBlockV1) && !p.RefineLeafDirections) || (p.RefineLeafDirections && !p.PreserveShared) {
		return errors.New("leaf refinement requires its explicit protocol and shared protection")
	}
	if p.Protocol == NMetricConstrainedV1 {
		if !p.PreserveShared || !slices.Contains([]string{"bias", "affine", "readout"}, p.Update) || (p.FixedBasis && (p.RefineLeafDirections || p.BlockSparse)) {
			return errors.New("invalid constrained subspace")
		}
	} else if nMetricReuse(p.Protocol) {
		if !p.FixedBasis || !p.PreserveShared || p.Update != "" || p.BlockSparse || p.RefineLeafDirections || p.Strategy != "full-refit" || nMetricFeatureLimit(p) != 32 {
			return errors.New("shared basis reuse requires fixed 32-feature readout-only search")
		}
	} else if p.Update != "" || p.FixedBasis || (p.BlockSparse && p.Protocol != NMetricBlockV1) {
		return errors.New("constrained settings require versioned protocol")
	}
	if len(p.CoordinateBlocks) > 32 || (len(p.CoordinateBlocks) > 0 && !p.BlockSparse) {
		return errors.New("invalid coordinate blocks")
	}
	for _, block := range p.CoordinateBlocks {
		if len(block) == 0 || len(block) > 7 {
			return errors.New("invalid block width")
		}
		seen := map[int]bool{}
		for _, j := range block {
			if j < 0 || j >= nDimensions || seen[j] {
				return errors.New("invalid block coordinate")
			}
			seen[j] = true
		}
	}
	if p.Protocol == NMetricBlockV1 && (!p.BlockSparse || !p.PreserveShared) {
		return errors.New("block refinement requires explicit shared protection")
	}
	if !slices.Contains([]string{"full-refit", "new-feature", "bounded-joint", "combined"}, p.Strategy) {
		return errors.New("invalid numerical proposal strategy")
	}
	if len(p.Fractions) > 4 || (p.Strategy != "full-refit" && len(p.Fractions) == 0) {
		return errors.New("bounded proposal requires one to four frozen fractions")
	}
	previous := 0.
	for _, a := range p.Fractions {
		if !nFinite(a) || a <= previous || a >= 1 {
			return errors.New("proposal fractions must increase strictly in (0,1)")
		}
		previous = a
	}
	return nil
}

func nMetricHash(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func nMetricSameExperience(a, b NWitness) bool {
	// An episode cannot be both fit and check, even under another query.
	// Identical encoded input under the same query is also not independent.
	return a.EpisodeID == b.EpisodeID || (slices.Equal(a.Query, b.Query) && slices.Equal(a.X, b.X))
}

func nMetricIndependentEvidence(fit, check, witnesses []NWitness, query []NodeID) ([]NWitness, *NMetricEvidence, error) {
	for _, a := range fit {
		for _, b := range check {
			if nMetricSameExperience(a, b) {
				return nil, nil, errors.New("numerical check overlaps fit evidence")
			}
		}
	}
	out := slices.Clone(fit)
	e := &NMetricEvidence{Check: len(check), WitnessSHA256: nMetricHash(witnesses)}
	for _, w := range witnesses {
		if !slices.Equal(w.Query, query) {
			continue
		}
		overlap := slices.ContainsFunc(check, func(c NWitness) bool { return nMetricSameExperience(w, c) })
		// New check witnesses remain retention-only, including on later lessons.
		// Legacy witnesses retain their original role except for current overlap.
		if w.Role == "evaluation" || overlap {
			e.WitnessExcluded++
			continue
		}
		if !slices.ContainsFunc(out, func(f NWitness) bool { return nMetricSameExperience(w, f) }) {
			out = append(out, w)
			e.WitnessFit++
		}
	}
	e.Fit = len(out)
	e.FitSHA256 = nMetricHash(out)
	e.CheckSHA256 = nMetricHash(check)
	return out, e, nil
}

type nMetricProposal struct {
	Model      nModel
	Name       string
	Fraction   float64
	Valid      bool
	Rejection  string
	Constraint *NConstraintTrace
}

func nMetricValidHead(m nModel, hi int) bool {
	h := m.Heads[hi]
	if !nFinite(h.B) || math.Abs(h.B) > 7.99 {
		return false
	}
	for _, w := range h.W {
		if !nFinite(w) || math.Abs(w) > 7.99 {
			return false
		}
	}
	return true
}

func nMetricProposals(before, full nModel, query []NodeID, fit []NWitness, p *NMetricProposalConfig, fullValid bool) []nMetricProposal {
	if p == nil {
		return []nMetricProposal{{Model: full, Valid: fullValid}}
	}
	hi, bi := nHeadIndex(full, query), nHeadIndex(before, query)
	out := []nMetricProposal{}
	if p.PreserveShared && nMetricSharedChange(before, full, query) {
		// Keep the proposal in the audit. This task-local refit has no evidence
		// to revise another head's representation; query weights stay plastic.
		return []nMetricProposal{{Model: full, Name: "full-refit", Fraction: 1, Valid: fullValid, Rejection: "cross-query-representation-change"}}
	}
	if p.Strategy == "full-refit" || p.Strategy == "combined" || p.Strategy == "bounded-joint" {
		out = append(out, nMetricProposal{Model: full, Name: "full-refit", Fraction: 1, Valid: fullValid})
	}
	// Interpolating coefficients from different readout bases has no meaning.
	// Resolution changes may still be tested by the unchanged full-refit arm.
	compatible := bi >= 0 && before.Heads[bi].Known && before.Heads[bi].Metric == full.Heads[hi].Metric && len(full.Circuits) >= len(before.Circuits)
	if compatible {
		for i, c := range before.Circuits {
			compatible = compatible && reflect.DeepEqual(c, full.Circuits[i])
		}
	}
	if !compatible {
		return out
	}
	inc := before.Heads[bi]
	if fullValid && (p.Strategy == "bounded-joint" || p.Strategy == "combined") {
		for _, a := range p.Fractions {
			m := nClone(full)
			h := &m.Heads[hi]
			h.B = inc.B + a*(h.B-inc.B)
			for i, w := range h.W {
				old := 0.
				if i < len(inc.W) {
					old = inc.W[i]
				}
				h.W[i] = old + a*(w-old)
			}
			out = append(out, nMetricProposal{Model: m, Name: "bounded-joint", Fraction: a, Valid: nMetricValidHead(m, hi)})
		}
	}
	if len(full.Circuits) != len(before.Circuits)+1 || (p.Strategy != "new-feature" && p.Strategy != "combined") {
		return out
	}
	// Fit just the new coefficient to the incumbent residual. Old bias and
	// weights stay exactly fixed for this proposal, not permanently frozen.
	numerator, denominator := 0., 0.
	for _, r := range fit {
		v := nMetricValues(full, r.X, inc.Metric)
		x := v[len(v)-1]
		numerator += x * (r.Outcome - nPredict(before, r.X, query))
		denominator += x * x
	}
	if denominator <= 0 || !nFinite(numerator/denominator) {
		return out
	}
	w := numerator / denominator
	for _, a := range append(slices.Clone(p.Fractions), 1) {
		m := nClone(full)
		h := &m.Heads[hi]
		h.B = inc.B
		h.W = append(slices.Clone(inc.W), a*w)
		h.Known = true
		out = append(out, nMetricProposal{Model: m, Name: "new-feature", Fraction: a, Valid: nMetricValidHead(m, hi)})
	}
	return out
}

func nMetricSharedChange(before, after nModel, query []NodeID) bool {
	used := map[NodeID]bool{}
	var include func(NodeID)
	include = func(id NodeID) {
		if used[id] {
			return
		}
		used[id] = true
		for _, c := range before.Circuits {
			if c.Meta.Node == id {
				for _, parent := range c.Meta.Parents {
					include(parent)
				}
			}
		}
	}
	for _, h := range before.Heads {
		if slices.Equal(h.Query, query) || !h.Known {
			continue
		}
		for i, w := range h.W {
			if w != 0 {
				include(before.Circuits[i].Meta.Node)
			}
		}
	}
	for i, c := range before.Circuits {
		if used[c.Meta.Node] && (i >= len(after.Circuits) || !reflect.DeepEqual(c, after.Circuits[i])) {
			return true
		}
	}
	return false
}
