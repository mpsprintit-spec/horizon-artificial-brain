package runtime

import (
	"bytes"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	"testing"
)

func TestNativeObservedEnvironmentGainAndReplay(t *testing.T) {
	r, ids := nativeFixture(t)
	dir := t.TempDir()
	w, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	s1Check(t, e)
	defer w.Close()
	r.SetEventLog(w)
	cfg := nativeControlConfig(ids)
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "observed", Observation: &NativeObservation{Source: "environment", SampleID: "observed", State: map[k.NodeID]float64{ids[0]: 1}}}})
	cp := filepath.Join(dir, "source.json")
	s1Check(t, r.Checkpoint(cp))
	gain, action := .4, 0
	result := nativeApply(t, r, NativeControlCommand{ID: "actual", Step: &NativeStep{EpisodeID: "observed", ActionOverride: &action, EnvironmentGain: &gain}})
	if math.Abs(result.State[ids[0]]-.6) > 1e-12 || math.Abs(result.State[ids[1]]-.4) > 1e-12 {
		t.Fatal("gain did not affect actual recurrent operation", result.State)
	}
	recovered, e := RebuildRuntimeFromCheckpointAndJournal(w, cp)
	s1Check(t, e)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, recovered)) {
		t.Fatal("WAL15 exact suffix replay failed")
	}
	events, e := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	s1Check(t, e)
	last := events[len(events)-1]
	if last.SchemaVersion != 15 {
		t.Fatal("environment field lacks WAL15 guard")
	}
	last.SchemaVersion = 14
	if validateJournalPayload(last) == nil {
		t.Fatal("downgraded environment command accepted")
	}
	last = events[0]
	last.SchemaVersion = 15
	if validateJournalPayload(last) == nil {
		t.Fatal("WAL15 accepted without environment condition")
	}
	before := continuationBytes(t, r)
	for _, bad := range []float64{0, -.1, 1.01, math.Inf(1), math.NaN()} {
		_, e := r.ApplyNativeControl(NativeControlCommand{ID: "bad", Step: &NativeStep{EpisodeID: "observed", ActionOverride: &action, EnvironmentGain: &bad}}, s1Time(r))
		if e == nil {
			t.Fatal("invalid environmental gain accepted")
		}
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("rejected input mutated Brain")
	}
}

func TestMetricAuditedAndRefinementCheckpointReplay(t *testing.T) {
	for _, protocol := range []string{k.NMetricAuditV1, k.NMetricRefineV1} {
		t.Run(protocol, func(t *testing.T) {
			r, wal, dir, start := mNew(t, k.NeuralCompositionMetricV6)
			e := nLesson(0, 0, nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), func(p nXY) float64 { return .3 * p.x })
			e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: protocol, Strategy: "combined", Fractions: []float64{.25, .5}, PreserveShared: true, RefineLeafDirections: protocol == k.NMetricRefineV1}
			if !mTeach(t, r, e, 1e-5).Committed {
				t.Fatal("audited lesson did not commit")
			}
			cp := filepath.Join(dir, "audited.json")
			s1Check(t, r.Checkpoint(cp))
			expected := continuationBytes(t, r)
			for _, source := range []string{cp, start} {
				got, err := RebuildRuntimeFromCheckpointAndJournal(wal, source)
				s1Check(t, err)
				if !bytes.Equal(expected, continuationBytes(t, got)) {
					t.Fatal("audited exact recovery failed")
				}
			}
		})
	}
}
