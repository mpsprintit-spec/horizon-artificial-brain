//go:build horizon_s2r1

package runtime

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func s2r1Transform(e k.CompositionExposure, witness []k.NWitness, name string) (k.CompositionExposure, []k.NWitness) {
	b, _ := json.Marshal(e)
	var result k.CompositionExposure
	_ = json.Unmarshal(b, &result)
	b, _ = json.Marshal(witness)
	var w []k.NWitness
	_ = json.Unmarshal(b, &w)
	extra := func(in k.CompositionInput) []float64 {
		out := []float64{}
		for _, j := range []int{56, 57, 58, 59, 60, 61, 63, 64} {
			out = append(out, in.Sequence[len(in.Sequence)-1-j/7][j%7])
		}
		return out
	}
	byID := map[string][]float64{}
	for _, rows := range [][]k.NObservation{e.Resolution.Fit, e.Resolution.Check} {
		for _, r := range rows {
			byID[r.EpisodeID] = extra(r.Input)
		}
	}
	keys := []string{}
	for id := range byID {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	permuted := map[string][]float64{}
	order := rand.New(rand.NewSource(3203)).Perm(len(keys))
	for i, id := range keys {
		permuted[id] = byID[keys[order[i]]]
	}
	change := func(v []float64, id string) []float64 {
		v = slices.Clone(v)
		switch name {
		case "B-no-forecast":
			for j := 0; j < 3; j++ {
				v[j] = 0
			}
		case "C-no-history":
			for j := 3; j < 8; j++ {
				v[j] = 0
			}
		case "D-permuted-extra":
			if other, ok := permuted[id]; ok {
				v = slices.Clone(other)
			}
		case "E-history-scale4":
			for j := 3; j < 6; j++ {
				v[j] *= 4
			}
		}
		return v
	}
	for _, rows := range [][]k.NObservation{result.Resolution.Fit, result.Resolution.Check} {
		for i := range rows {
			v := change(extra(rows[i].Input), rows[i].EpisodeID)
			for p, j := range []int{56, 57, 58, 59, 60, 61, 63, 64} {
				rows[i].Input.Sequence[len(rows[i].Input.Sequence)-1-j/7][j%7] = v[p]
			}
		}
	}
	for i := range w {
		v := []float64{}
		for _, j := range []int{56, 57, 58, 59, 60, 61, 63, 64} {
			v = append(v, w[i].X[j])
		}
		v = change(v, w[i].EpisodeID)
		for p, j := range []int{56, 57, 58, 59, 60, 61, 63, 64} {
			w[i].X[j] = v[p]
		}
	}
	result.Input = result.Resolution.Fit[0].Input
	return result, w
}

func TestS2R1Followup(t *testing.T) {
	root, work := s2r1Paths(t)
	events, e := ReadEventLog(filepath.Join(work, "source-events.jsonl"))
	s1Check(t, e)
	commands := map[int]LoggedEvent{}
	n := 0
	for _, event := range events {
		if event.CompositionExposure != nil && len(event.CompositionExposure.ID) >= 18 && event.CompositionExposure.ID[:18] == "s2-error-learning-" {
			n++
			commands[n] = event
		}
	}
	var receipt []k.NDiagnosis
	s1Read(t, filepath.Join(root, "s2/learning.json"), &receipt)
	r := NewBrainRuntime(k.NewBootstrapBrain())
	s1Check(t, r.RestoreCheckpoint(filepath.Join(root, "s2/02-stage-08.json")))
	r.SetClock(NewFixedClock(s1Base))
	before := continuationBytes(t, r)
	original := r.brain.BrainState.NResolution.Witnesses
	conditions, controls, steps := []map[string]any{}, []map[string]any{}, []map[string]any{}
	for _, lesson := range []int{5, 9} {
		event := commands[lesson]
		for _, name := range []string{"A-original", "B-no-forecast", "C-no-history", "D-permuted-extra", "E-history-scale4"} {
			input, w := s2r1Transform(*event.CompositionExposure, original, name)
			r.brain.BrainState.NResolution.Witnesses = w
			tr, err := r.brain.S2R1Diagnose(r.composition.Config, input, event.Timestamp)
			r.brain.BrainState.NResolution.Witnesses = original
			s1Check(t, err)
			if name == "A-original" {
				a, _ := json.Marshal(tr.Diagnosis.Trials)
				b, _ := json.Marshal(receipt[lesson-1].Trials)
				if !bytes.Equal(a, b) {
					t.Fatal("stage08 state is not an equivalent numerical control", lesson)
				}
				steps = append(steps, map[string]any{"lesson": lesson, "probes": tr.S2R1PartialStepProbes()})
			}
			conditions = append(conditions, map[string]any{"lesson": lesson, "condition": name, "trace": tr})
		}
		for _, name := range []string{"remove-S2-witnesses", "retain-only-S2-witnesses"} {
			w := []k.NWitness{}
			for _, v := range original {
				same := reflect.DeepEqual(v.Query, receipt[lesson-1].Query)
				if (name == "remove-S2-witnesses" && !same) || (name == "retain-only-S2-witnesses" && same) {
					w = append(w, v)
				}
			}
			r.brain.BrainState.NResolution.Witnesses = w
			tr, err := r.brain.S2R1Diagnose(r.composition.Config, *event.CompositionExposure, event.Timestamp)
			r.brain.BrainState.NResolution.Witnesses = original
			s1Check(t, err)
			controls = append(controls, map[string]any{"lesson": lesson, "control": name, "trace": tr, "selected_vs_original_retention": tr.S2R1OriginalRetention(original)})
		}
		t.Log("completed equivalent input, witness and parameter controls for historical lesson", lesson)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("controlled interventions did not restore exact diagnostic parent")
	}
	s2r1Write(t, work, "learning-input-interventions.json", conditions)
	s2r1Write(t, work, "witness-controls.json", controls)
	s2r1Write(t, work, "partial-step-interventions.json", steps)
	t.Log("no installation, no new learning command; original stage08 runtime bytes restored exactly")
}

func TestS2R1RankedTail(t *testing.T) {
	root, work := s2r1Paths(t)
	events, e := ReadEventLog(filepath.Join(work, "source-events.jsonl"))
	s1Check(t, e)
	r := NewBrainRuntime(k.NewBootstrapBrain())
	s1Check(t, r.RestoreCheckpoint(filepath.Join(root, "s2/02-stage-08.json")))
	r.SetClock(NewFixedClock(s1Base))
	before := continuationBytes(t, r)
	output := []map[string]any{}
	n := 0
	for _, event := range events {
		if event.CompositionExposure == nil || len(event.CompositionExposure.ID) < 18 || event.CompositionExposure.ID[:18] != "s2-error-learning-" {
			continue
		}
		n++
		if n != 5 && n != 9 {
			continue
		}
		tr, err := r.brain.S2R1Diagnose(r.composition.Config, *event.CompositionExposure, event.Timestamp)
		s1Check(t, err)
		output = append(output, map[string]any{"lesson": n, "native_ranking_n": len(tr.Ranking), "first_rank": 9, "last_rank": 64, "probes": tr.S2R1TailProbes(9, 64)})
		t.Log("completed fixed ranks 9-64 without installation for lesson", n)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("ranked candidate counterfactual mutated state")
	}
	s2r1Write(t, work, "ranked-tail-probes.json", output)
}

func TestS2R1RecruitmentCounterfactual(t *testing.T) {
	root, work := s2r1Paths(t)
	events, e := ReadEventLog(filepath.Join(work, "source-events.jsonl"))
	s1Check(t, e)
	r := NewBrainRuntime(k.NewBootstrapBrain())
	s1Check(t, r.RestoreCheckpoint(filepath.Join(root, "s2/02-stage-08.json")))
	r.SetClock(NewFixedClock(s1Base))
	before := continuationBytes(t, r)
	for _, event := range events {
		if event.CompositionExposure == nil || event.CompositionExposure.ID != "s2-error-learning-09" {
			continue
		}
		tr, err := r.brain.S2R1Diagnose(r.composition.Config, *event.CompositionExposure, event.Timestamp)
		s1Check(t, err)
		s2r1Write(t, work, "direction-step-contributions.json", tr.S2R1DirectionStepContribution())
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("direction contribution diagnostic mutated runtime")
	}
	t.Log("split old-readout/new-feature changes and extra-coordinate interventions without installing a candidate")
}
