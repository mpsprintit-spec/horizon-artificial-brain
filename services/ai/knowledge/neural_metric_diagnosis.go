package knowledge

import (
	"math"
	"time"
)

// Requests are bounded hypotheses for an observer/planner, not claims that a
// physical cause was discovered. They never authorize an external action.
func (k *KnowledgeBase) finishNMetricDiagnosis(d NDiagnosis, tolerance *float64, bound float64, at time.Time) {
	if tolerance != nil {
		v := *tolerance
		d.RequiredPrecision = &v
	}
	if !nFinite(bound) {
		bound = 2
	}
	bound = math.Min(2, math.Max(0, bound))
	d.EstimatedAbsError = &bound
	k.BrainState.NResolution.Last = d
	k.recordNDiagnosis(d, at)
	k.BrainState.InquiryState.Pending = len(d.Probes) > 0
	k.BrainState.CuriosityState["n_observed_uncertainty"] = bound
	if tolerance != nil {
		k.BrainState.CuriosityState["n_precision_gap"] = math.Max(0, bound-*tolerance)
	}
}
