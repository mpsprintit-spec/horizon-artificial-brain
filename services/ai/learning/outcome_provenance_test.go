package learning

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func provenanceOutcome(request, source, group, cause, episode string) OutcomeEvidence {
	return OutcomeEvidence{RequestID: request, NodeID: 1, Source: source, IndependenceGroup: group, CausalGroup: cause, EpisodeID: episode, Evidence: Evidence{Weight: .6, Confidence: .5, Reliability: .9, IndependentSources: 999}}
}

func TestOutcomeProvenanceSeparatesMeasurementsFromIndependentSupport(t *testing.T) {
	tests := []struct {
		name       string
		records    []OutcomeEvidence
		want       knowledge.EvidenceCounts
		confidence float64
	}{
		{"same source new requests and episodes", []OutcomeEvidence{provenanceOutcome("r1", "camera", "g1", "c1", "e1"), provenanceOutcome("r2", "camera", "g2", "c1", "e2")}, knowledge.EvidenceCounts{Observations: 2, Episodes: 2, Sources: 1, CausalGroups: 1, IndependentGroups: 1}, .5},
		{"distinct sources shared cause", []OutcomeEvidence{provenanceOutcome("r1", "camera", "g1", "scene", "e"), provenanceOutcome("r2", "relay", "g2", "scene", "e")}, knowledge.EvidenceCounts{Observations: 2, Episodes: 1, Sources: 2, CausalGroups: 1, IndependentGroups: 1}, .5},
		{"distinct sources shared group", []OutcomeEvidence{provenanceOutcome("r1", "camera", "g", "", ""), provenanceOutcome("r2", "relay", "g", "", "")}, knowledge.EvidenceCounts{Observations: 2, Sources: 2, IndependentGroups: 1}, .5},
		{"undeclared independence", []OutcomeEvidence{provenanceOutcome("r1", "camera", "", "", ""), provenanceOutcome("r2", "other-camera", "", "", "")}, knowledge.EvidenceCounts{Observations: 2, Sources: 2}, .5},
		{"declared separate origins", []OutcomeEvidence{provenanceOutcome("r1", "camera", "g1", "c1", "e1"), provenanceOutcome("r2", "other-camera", "g2", "c2", "e2")}, knowledge.EvidenceCounts{Observations: 2, Episodes: 2, Sources: 2, CausalGroups: 2, IndependentGroups: 2}, .6},
		{"delimiter identities", []OutcomeEvidence{provenanceOutcome("a|b", "c", "g1", "", ""), provenanceOutcome("a", "b|c", "g2", "", "")}, knowledge.EvidenceCounts{Observations: 2, Sources: 2, IndependentGroups: 2}, .6},
		{"opaque request identities", []OutcomeEvidence{provenanceOutcome("r", "camera", "g", "", ""), provenanceOutcome(" r ", "camera", "g", "", "")}, knowledge.EvidenceCounts{Observations: 2, Sources: 1, IndependentGroups: 1}, .5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewEvidenceLedger()
			var e Evidence
			for _, o := range tt.records {
				var added bool
				var err error
				e, added, err = l.Record(o)
				if err != nil || !added {
					t.Fatalf("record: %v %v", added, err)
				}
			}
			if got := l.Counts(1); got != tt.want {
				t.Fatalf("counts got %+v want %+v", got, tt.want)
			}
			if e.IndependentSources != tt.want.IndependentGroups || e.Confidence != tt.confidence {
				t.Fatalf("aggregate: %+v", e)
			}
			wantPromotion := PromotionCandidate
			if tt.want.IndependentGroups >= 2 {
				wantPromotion = PromotionAccepted
			}
			if got := DefaultLearningPolicy().Evaluate(e); got != wantPromotion {
				t.Fatalf("promotion got %v want %v", got, wantPromotion)
			}
			before := l.Snapshot()
			retry := tt.records[0]
			retry.IndependenceGroup = "retry-must-not-rewrite-lineage"
			retry.Evidence.Contradictions = 1
			if _, added, err := l.Record(retry); err != nil || added {
				t.Fatalf("retry: %v %v", added, err)
			}
			if !reflect.DeepEqual(before, l.Snapshot()) {
				t.Fatal("retry mutated accepted evidence")
			}
			r, err := RestoreEvidenceLedger(l.Snapshot())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(l.Snapshot(), r.Snapshot()) || r.Counts(1) != tt.want {
				t.Fatal("restart changed evidence or lineage")
			}
		})
	}
}

func TestOutcomeProvenanceLateBridgeMergesSupportAndLegacyStaysUnclassified(t *testing.T) {
	l := NewEvidenceLedger()
	for _, o := range []OutcomeEvidence{provenanceOutcome("r1", "a", "g1", "c1", ""), provenanceOutcome("r2", "b", "g2", "c2", "")} {
		if _, _, err := l.Record(o); err != nil {
			t.Fatal(err)
		}
	}
	if l.Counts(1).IndependentGroups != 2 {
		t.Fatal("separate declared origins did not count")
	}
	e, _, err := l.Record(provenanceOutcome("r3", "a", "g1", "c2", ""))
	if err != nil {
		t.Fatal(err)
	}
	if e.IndependentSources != 1 || e.Confidence != .5 {
		t.Fatalf("late common-cause bridge did not reduce support: %+v", e)
	}
	legacy := NewEvidenceLedger()
	for _, request := range []string{"old1", "old2"} {
		o := provenanceOutcome(request, "camera", "", "", "")
		o.Evidence.IndependentSources = 1
		if _, _, err := legacy.RecordLegacy(o); err != nil {
			t.Fatal(err)
		}
	}
	if legacy.Snapshot().ByNode[1].IndependentSources != 2 {
		t.Fatal("historical law changed")
	}
	if _, added, err := legacy.Record(provenanceOutcome("old1", "camera", "invented", "", "")); err != nil || added {
		t.Fatal("adoption replayed an already applied old outcome")
	}
	e, _, err = legacy.Record(provenanceOutcome("new", "camera", "g", "cause", "e"))
	if err != nil {
		t.Fatal(err)
	}
	if e.IndependentSources != 1 || e.Confidence != .5 || legacy.Counts(1).LegacyObservations != 2 {
		t.Fatalf("legacy support was fabricated on adoption: %+v %+v", e, legacy.Counts(1))
	}
	r, err := RestoreEvidenceLedger(legacy.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(legacy.Snapshot(), r.Snapshot()) {
		t.Fatal("legacy lineage was lost")
	}
	if _, _, err = r.RecordLegacy(provenanceOutcome("downgrade", "camera", "", "", "")); err == nil {
		t.Fatal("silent downgrade accepted")
	}
}

func TestOutcomeProvenanceRejectsInvalidWithoutMutationAndOwnsSnapshots(t *testing.T) {
	l := NewEvidenceLedger()
	o := provenanceOutcome("r", "camera", "g", "c", "e")
	if _, _, err := l.Record(o); err != nil {
		t.Fatal(err)
	}
	before := l.Snapshot()
	for _, edit := range []func(*OutcomeEvidence){func(o *OutcomeEvidence) { o.Source = " " }, func(o *OutcomeEvidence) { o.RequestID = "" }, func(o *OutcomeEvidence) { o.NodeID = 0 }, func(o *OutcomeEvidence) { o.Evidence.Confidence = math.NaN() }, func(o *OutcomeEvidence) { o.Evidence.Reliability = math.Inf(1) }, func(o *OutcomeEvidence) { o.Evidence.Weight = -.1 }, func(o *OutcomeEvidence) { o.Evidence.Contradictions = -1 }} {
		bad := o
		edit(&bad)
		if _, _, err := l.Record(bad); err == nil {
			t.Fatal("invalid evidence accepted")
		}
		if !reflect.DeepEqual(before, l.Snapshot()) {
			t.Fatal("invalid record mutated ledger")
		}
	}
	r, err := RestoreEvidenceLedger(before)
	if err != nil {
		t.Fatal(err)
	}
	p := before.Provenance[1]
	p.Records[0].Source = "mutated"
	before.Provenance[1] = p
	if !reflect.DeepEqual(l.Snapshot(), r.Snapshot()) {
		t.Fatal("snapshot/restore shares typed data")
	}
}

func TestOutcomeProvenanceSnapshotRejectsInconsistentLineage(t *testing.T) {
	l := NewEvidenceLedger()
	if _, _, err := l.Record(provenanceOutcome("r", "camera", "g", "c", "e")); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*EvidenceLedgerSnapshot){
		func(s *EvidenceLedgerSnapshot) { e := s.ByNode[1]; e.IndependentSources = 99; s.ByNode[1] = e },
		func(s *EvidenceLedgerSnapshot) {
			p := s.Provenance[1]
			p.Records = append(p.Records, p.Records[0])
			s.Provenance[1] = p
		},
		func(s *EvidenceLedgerSnapshot) {
			p := s.Provenance[1]
			p.Records[0].Source = "different"
			s.Provenance[1] = p
		},
		func(s *EvidenceLedgerSnapshot) { p := s.Provenance[1]; p.Records[0].NodeID = 2; s.Provenance[1] = p },
		func(s *EvidenceLedgerSnapshot) {
			p := s.Provenance[1]
			p.LegacySeen = []string{"unknown"}
			s.Provenance[1] = p
		},
		func(s *EvidenceLedgerSnapshot) {
			p := s.Provenance[1]
			p.Records[0].Evidence.IndependentSources = 123
			s.Provenance[1] = p
		},
		func(s *EvidenceLedgerSnapshot) { s.Provenance[2] = s.Provenance[1] },
	} {
		s := l.Snapshot()
		edit(&s)
		if _, err := RestoreEvidenceLedger(s); err == nil {
			data, _ := json.Marshal(s)
			t.Fatalf("invalid snapshot accepted: %s", data)
		}
	}
}

func TestOutcomeProvenanceConcurrentRetriesRemainOneObservationEach(t *testing.T) {
	l := NewEvidenceLedger()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			o := provenanceOutcome(fmt.Sprint(i), fmt.Sprintf("source-%d", i), "shared", "same-cause", "same-episode")
			for j := 0; j < 10; j++ {
				if _, _, err := l.Record(o); err != nil {
					t.Error(err)
				}
			}
		}(i)
	}
	wg.Wait()
	want := knowledge.EvidenceCounts{Observations: 12, Episodes: 1, Sources: 12, CausalGroups: 1, IndependentGroups: 1}
	if l.Counts(1) != want {
		t.Fatalf("concurrent counts: %+v", l.Counts(1))
	}
	if _, err := RestoreEvidenceLedger(l.Snapshot()); err != nil {
		t.Fatal(err)
	}
}
