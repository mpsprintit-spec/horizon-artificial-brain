//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestS2R7Prepare(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	cp, wal := s2r3Source(root)
	source := orderedFixtureClone(t, cp)
	contract := filepath.Join(work, "history-contract")
	for i, name := range []string{"anchors", "coverage/normal", "coverage/shifted", "retention-development/normal", "retention-development/shifted"} {
		dir := filepath.Join(contract, name)
		r, w := s2r2Fork(t, cp, wal, dir)
		t.Setenv("HORIZON_S2_WORK", dir)
		n := 64
		if i >= 3 {
			n = 128
		}
		rows := s2r5OldCollect(t, r, pop, 271810001+int64(i*1000), n, i == 2 || i == 4, nil, -1)
		s2r7Write(t, filepath.Join(dir, "rows.json"), rows)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
		if i >= 3 {
			s2r7Write(t, filepath.Join(dir, "source-metrics.json"), s2r4Retention(t, source, source, rows, pop))
		}
	}
	s2r7Write(t, filepath.Join(work, "R7_RETENTION_CONTRACT.json"), map[string]any{"historical_RMSE_limit": .01766457, "S1_shift": 0, "paired_ratio_upper_CI95_limit": 1.05, "R6_contract_unchanged": true, "source_RNG_family": 271810001, "new_contract_is_separate": true})
}

func s2r7PhaseSeed(t *testing.T) (string, int, int) {
	phase := os.Getenv("HORIZON_S2R7_PHASE")
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R7_SEED"))
	s1Check(t, e)
	if !slices.Contains(s2r5Seeds, seed) {
		t.Fatal("unregistered seed")
	}
	n := 2048
	if phase == "confirmatory" {
		n, e = strconv.Atoi(os.Getenv("HORIZON_S2R7_BUDGET"))
		s1Check(t, e)
		if n != 512 && n != 1024 && n != 2048 {
			t.Fatal("unfrozen budget")
		}
	}
	if phase != "development" && phase != "confirmatory" {
		t.Fatal("unknown phase")
	}
	return phase, seed, n
}

func s2r7Bootstrap(t *testing.T, root, work, base string, pop [][]k.NodeID, seed int, phase string) *BrainRuntime {
	cp, wal := s2r3Source(root)
	rng := int64(271000000 + 1000*seed)
	if phase == "confirmatory" {
		rng += 10000000
	}
	od := filepath.Join(base, "history-observer")
	observer, ow := s2r2Fork(t, cp, wal, od)
	rows := s2r4Collect(t, observer, pop, od, "persistent", rng, 16, 16, false)
	s1Check(t, ow.Close())
	for _, masked := range []bool{false, true} {
		name := "bootstrap"
		if masked {
			name = "history-masked"
		}
		dir := filepath.Join(base, name)
		r, w := s2r2Fork(t, cp, wal, dir)
		s2r5ReplayObserver(t, r, filepath.Join(od, "events.jsonl"))
		receipts := []*CompositionLearningReceipt{}
		for b := 0; b < 4; b++ {
			rec, _ := s2r6HistoryLesson(t, r, s2r6Arm{Mask: masked, Coverage: true}, fmt.Sprintf("r7-%s-%d-%s-%d", phase, seed, name, b), s2r4Subset(rows, 4*b, 4*b+2), s2r4Subset(rows, 4*b+2, 4*b+4), pop, filepath.Join(work, "history-contract"))
			receipts = append(receipts, rec)
			s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
			t.Log("history", seed, name, b, rec.Diagnosis.Committed)
		}
		s2r7Write(t, filepath.Join(dir, "receipts.json"), receipts)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
	return orderedFixtureClone(t, filepath.Join(base, "bootstrap/final.json"))
}

func s2r7Partition(rows []s2r7Row, curriculum bool, seed int64) ([]s2r7Row, []s2r7Row) {
	streams := []string{}
	for _, r := range rows {
		if !slices.Contains(streams, r.Stream) {
			streams = append(streams, r.Stream)
		}
	}
	fit, check := []s2r7Row{}, []s2r7Row{}
	for _, r := range rows {
		if slices.Index(streams, r.Stream)%4 < 2 {
			fit = append(fit, r)
		} else {
			check = append(check, r)
		}
	}
	for si, set := range [][]s2r7Row{fit, check} {
		groups := [][]s2r7Row{}
		for i := 0; i < len(set); i += 2 {
			groups = append(groups, slices.Clone(set[i:i+2]))
		}
		if curriculum {
			slices.SortStableFunc(groups, func(a, b []s2r7Row) int { return a[0].Stage - b[0].Stage })
		} else {
			rng := rand.New(rand.NewSource(seed + int64(si)))
			rng.Shuffle(len(groups), func(i, j int) { groups[i], groups[j] = groups[j], groups[i] })
		}
		at := 0
		for _, group := range groups {
			copy(set[at:], group)
			at += len(group)
		}
	}
	return fit, check
}

func s2r7Lesson(t *testing.T, r *BrainRuntime, id string, fit, check []s2r7Row, regularized, shuffle bool, target string, seed int64) (*CompositionLearningReceipt, map[string]any) {
	t.Helper()
	at := s2r4Now(r)
	for _, set := range [][]s2r7Row{fit, check} {
		for _, row := range set {
			if row.ObservedAt.After(at) {
				at = row.ObservedAt
			}
			if row.Input.Scene.ObservedAt.After(at) {
				at = row.Input.Scene.ObservedAt
			}
		}
	}
	at = at.Add(time.Duration(r.LastSequence()) * time.Nanosecond)
	e := k.CompositionExposure{ID: id, Source: "r7-actual-native-paired-experience", EpisodeID: id, IndependenceGroup: id, CausalGroup: "native-operation-origin", Reliability: 1, Motor: []float64{0, 0, 0, 0}, After: s1Scene(at), Resolution: &k.NResolutionLesson{Proposal: &k.NMetricProposalConfig{Protocol: k.NMetricReuseV2, Strategy: "full-refit", FixedBasis: true, PreserveShared: true, FeatureLimit: 32}}}
	if regularized {
		e.Resolution.Proposal.Protocol = k.NMetricReuseV3
		e.Resolution.Proposal.Ridge = .01
	}
	for si, rows := range [][]s2r7Row{fit, check} {
		ys := make([]float64, len(rows))
		for i, row := range rows {
			ys[i] = row.Utility
			if target == "contribution" {
				ys[i] = row.Contribution / 2
			}
		}
		if shuffle {
			rng := rand.New(rand.NewSource(seed + int64(si)))
			rng.Shuffle(len(ys), func(i, j int) { ys[i], ys[j] = ys[j], ys[i] })
		}
		for i, row := range rows {
			in := row.Input
			if target == "retrospective" {
				in = s2r7Input(row.Context, row.Original, row.State, row.Descriptor, &row.Actual)
			}
			if target == "contribution" {
				in.Words = [][]float64{slices.Clone(in.Words[0])}
				in.Words[0][4] = 0
				in.Words[0][6] = 1
			}
			o := k.NObservation{Input: in, Outcome: ys[i], ObservedAt: row.ObservedAt, EpisodeID: row.OriginID}
			if si == 0 {
				e.Resolution.Fit = append(e.Resolution.Fit, o)
			} else {
				e.Resolution.Check = append(e.Resolution.Check, o)
			}
		}
	}
	e.Input = e.Resolution.Fit[0].Input
	start := time.Now()
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, at)
	s1Check(t, err)
	if len(r.brain.BrainState.NResolution.Features) != 32 {
		t.Fatal("feature-cap changed")
	}
	return receipt, map[string]any{"learning_nanoseconds": time.Since(start).Nanoseconds(), "memory": s2r3Memory(), "inventory": s2r3Inventory(t, r), "candidate_trials": len(receipt.Diagnosis.Trials)}
}

func s2r7ReadRows(t *testing.T, r *BrainRuntime, rows []s2r7Row, mode string) []float64 {
	pred := make([]float64, len(rows))
	for i, row := range rows {
		in := row.Input
		if mode == "no-history" {
			ctx := row.Context
			in = s2r7Input(ctx, row.Original, row.State, "D2", nil)
		}
		if mode == "retrospective" {
			in = s2r7Input(row.Context, row.Original, row.State, row.Descriptor, &row.Actual)
		}
		pred[i] = s2r6Read(t, r, in)
	}
	return pred
}

func TestS2R7Train(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	phase, seed, budget := s2r7PhaseSeed(t)
	base := filepath.Join(work, phase, fmt.Sprintf("seed-%d", seed))
	cp, wal := s2r3Source(root)
	if phase == "confirmatory" {
		if _, e := os.Stat(filepath.Join(work, "TRAINING_LOCK.json")); e != nil {
			t.Fatal("confirmatory configuration not frozen")
		}
	}
	brain := s2r7Bootstrap(t, root, work, base, pop, seed, phase)
	rng := int64(271300000 + 1000*seed)
	if phase == "confirmatory" {
		rng += 10000000
	}
	ob, ow := s2r2Fork(t, cp, wal, filepath.Join(base, "utility-observer"))
	descs := map[string]string{"D0": "D0", "D1": "D1", "D2": "D2", "D3": "D3", "D4": "D4"}
	readers := map[string]*BrainRuntime{}
	for d := range descs {
		readers[d] = brain
	}
	corpus, _ := s2r7Acquire(t, ob, readers, pop, filepath.Join(base, "utility-acquisition"), rng, budget/32, 16, false, false, descs, true)
	s1Check(t, ow.Close())
	ao, aw := s2r2Fork(t, cp, wal, filepath.Join(base, "auxiliary-observer"))
	aux, _ := s2r7Acquire(t, ao, map[string]*BrainRuntime{"D3": brain}, pop, filepath.Join(base, "auxiliary-acquisition"), rng+900000, 8, 16, false, false, map[string]string{"D3": "D3"}, true)
	s1Check(t, aw.Close())
	var dev map[string][]s2r7Row
	if phase == "development" {
		er, ew := s2r2Fork(t, cp, wal, filepath.Join(base, "development-observer"))
		dev, _ = s2r7Acquire(t, er, readers, pop, filepath.Join(base, "development-acquisition"), rng+1800000, 16, 16, false, false, descs, false)
		s1Check(t, ew.Close())
	}
	for _, arm := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		descriptor := "D3"
		if arm == "A" {
			descriptor = "D0"
		}
		if arm == "B" {
			descriptor = "D1"
		}
		dir := filepath.Join(base, arm)
		r, w := s2r2Fork(t, filepath.Join(base, "bootstrap/final.json"), filepath.Join(base, "bootstrap/events.jsonl"), dir)
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		curves := map[string]any{}
		regularized := arm == "D" || arm == "E" || arm == "F"
		if regularized {
			af, ac := s2r7Partition(aux["D3"], false, rng+13)
			targets := []string{"contribution"}
			if phase == "development" {
				targets = append(targets, "retrospective")
			}
			for i, target := range targets {
				rec, cost := s2r7Lesson(t, r, fmt.Sprintf("r7-%s-%d-%s-aux-%s", phase, seed, arm, target), af[64*i:64*(i+1)], ac[64*i:64*(i+1)], true, arm == "F", target, rng+int64(i))
				receipts = append(receipts, rec)
				costs = append(costs, cost)
				s2r3SaveCP(t, r, filepath.Join(dir, "aux-"+target+".json"))
			}
		}
		fit, check := s2r7Partition(corpus[descriptor], arm == "E", rng+111)
		fitIDs, checkIDs := []string{}, []string{}
		for _, row := range fit {
			fitIDs = append(fitIDs, row.ID)
		}
		for _, row := range check {
			checkIDs = append(checkIDs, row.ID)
		}
		s2r7Write(t, filepath.Join(dir, "TRAINING_ORDER.json"), map[string]any{"Fit": fitIDs, "Check": checkIDs, "origin_grouped": true, "descriptor": descriptor, "utility_budget": budget})
		if arm != "G" {
			for b := 0; b < budget/128; b++ {
				rec, cost := s2r7Lesson(t, r, fmt.Sprintf("r7-%s-%d-%s-b%d", phase, seed, arm, b), fit[64*b:64*b+64], check[64*b:64*b+64], regularized, arm == "F", "utility", rng+int64(100*b))
				receipts = append(receipts, rec)
				costs = append(costs, cost)
				s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
				if phase == "development" && slices.Contains([]int{128, 256, 512, 1024, 2048}, (b+1)*128) {
					curves[strconv.Itoa((b+1)*128)] = map[string]any{"prediction": s2r7ReadRows(t, r, dev[descriptor], "full"), "no-history": s2r7ReadRows(t, r, dev[descriptor], "no-history"), "retrospective": s2r7ReadRows(t, r, dev[descriptor], "retrospective")}
				}
				t.Log(phase, seed, arm, b, rec.Diagnosis.Committed, rec.Diagnosis.Cause)
			}
		}
		if regularized && phase != "development" {
			af, ac := s2r7Partition(aux["D3"], false, rng+13)
			rec, cost := s2r7Lesson(t, r, fmt.Sprintf("r7-%s-%d-%s-aux-retrospective", phase, seed, arm), af[64:128], ac[64:128], true, arm == "F", "retrospective", rng+1)
			receipts = append(receipts, rec)
			costs = append(costs, cost)
			s2r3SaveCP(t, r, filepath.Join(dir, "aux-retrospective.json"))
		}
		s2r7Write(t, filepath.Join(dir, "receipts.json"), receipts)
		s2r7Write(t, filepath.Join(dir, "costs.json"), costs)
		s2r7Write(t, filepath.Join(dir, "curves.json"), curves)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
}
