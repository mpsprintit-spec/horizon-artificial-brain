//go:build horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"
)

func s2r6Paths(t *testing.T) (string, string, [][]k.NodeID) {
	t.Helper()
	root, work := os.Getenv("HORIZON_S2R6_SOURCE"), os.Getenv("HORIZON_S2R6_WORK")
	if root == "" || work == "" {
		t.Skip("isolated authenticated R6 research only")
	}
	if root == work || filepath.Dir(work) == root {
		t.Fatal("overlapping source")
	}
	t.Setenv("HORIZON_S2R5_SOURCE", root)
	t.Setenv("HORIZON_S2R5_WORK", work)
	t.Setenv("HORIZON_S2_SOURCE", root)
	t.Setenv("HORIZON_S2_WORK", work)
	return root, work, s1Pop(t, filepath.Join(root, "source/source/source"))
}

func TestS2R6Prepare(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatal("work must be new")
	}
	s1Check(t, os.MkdirAll(work, 0700))
	s := s1Snapshot(t, cp)
	sum, err := checkpointChecksumForSchema(s)
	s1Check(t, err)
	if sum != s2r3Checksum || s.Checksum != sum || s.Sequence != 80735 {
		t.Fatal("BLOCKED source")
	}
	r, w := s2r2Fork(t, cp, wal, filepath.Join(work, "source-verification"))
	defer w.Close()
	inv := s2r3Inventory(t, r)
	if inv["neurons"] != 274 || inv["synapses"] != 6408 || inv["features"] != 32 || fmt.Sprint(inv["native_policy_updates"]) != "10752" {
		t.Fatal(inv)
	}
	if r.activation.Memory != r.brain || r.learning.Kb != r.brain || r.dnf.Brain() != r.brain {
		t.Fatal("multiple ownership")
	}
	replay, err := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(root, "s2r2/s2r2-final.json"))
	s1Check(t, err)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replay)) {
		t.Fatal("historical replay")
	}
	s2r2Copy(t, cp, filepath.Join(work, "00-source.json"))
	s2r2Copy(t, wal, filepath.Join(work, "source-events.jsonl"))
	s1Write(t, filepath.Join(work, "SOURCE_VERIFICATION.json"), map[string]any{"sequence": 80735, "native_checksum": sum, "checkpoint_sha256": s2r3FileHash(t, cp), "journal_sha256": s2r3FileHash(t, wal), "inventory": inv, "historical_replay_exact": true, "single_owner": true})
	s1Write(t, filepath.Join(work, "ORIGINAL_WITNESSES.json"), r.brain.BrainState.NResolution.Witnesses)
	// Fresh normal anchors are evidence only; none is an R5 failed holdout.
	ad := filepath.Join(work, "anchors")
	ar, aw := s2r2Fork(t, cp, wal, ad)
	t.Setenv("HORIZON_S2_WORK", ad)
	anchors := s2r5OldCollect(t, ar, pop, 171810001, 64, false, nil, -1)
	s1Write(t, filepath.Join(ad, "rows.json"), anchors)
	s2r3SaveCP(t, ar, filepath.Join(ad, "final.json"))
	s1Check(t, aw.Close())
}

type s2r6UtilityRow struct {
	ID, OriginID, Stream           string
	Transfer                       bool
	Intervention                   k.S2R6Intervention
	Context                        k.S2R6Context
	Input                          k.CompositionInput
	Before, After, Actual, Utility float64
	Prediction                     float64
	PredictionNanoseconds          int64
	BeforeDigest, AfterDigest      string
	ObservedAt                     time.Time
}

func s2r6UtilityInput(ctx k.S2R6Context, at time.Time, query int) k.CompositionInput {
	word := make([]float64, 16)
	word[query] = 1
	word[15] = .125
	seq := [][]float64{}
	for i := 0; i < len(ctx.Measurements); i += 6 {
		row := make([]float64, 7)
		copy(row, ctx.Measurements[i:min(i+6, len(ctx.Measurements))])
		row[6] = 1
		seq = append(seq, row)
	}
	precision := .00001
	return k.CompositionInput{Words: [][]float64{word}, Scene: s1Scene(at), Sequence: seq, RequiredPrecision: &precision}
}

func s2r6Read(t *testing.T, r *BrainRuntime, in k.CompositionInput) float64 {
	t.Helper()
	o, err := r.ReadComposition(in, in.Scene.ObservedAt)
	s1Check(t, err)
	if o.NResolution == nil {
		return 0
	}
	return o.NResolution.Value
}

func s2r6CausalRows(t *testing.T, r *BrainRuntime, rows []s2r4Row, pop [][]k.NodeID, dir string, seed int64, transfer bool) []s2r6UtilityRow {
	t.Helper()
	s1Check(t, os.MkdirAll(dir, 0700))
	f, err := os.OpenFile(filepath.Join(dir, "preintervention.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, err)
	defer f.Close()
	rng := rand.New(rand.NewSource(seed))
	out := []s2r6UtilityRow{}
	features := []int{}
	for i := 0; i < 32; i++ {
		if (i%3 == 0) == transfer {
			features = append(features, i)
		}
	}
	for i, row := range rows {
		for trial := 0; trial < 2; trial++ {
			idx := features[rng.Intn(len(features))]
			ids := []int{idx}
			if (i+trial)%4 == 0 {
				other := features[rng.Intn(len(features))]
				if other != idx {
					ids = append(ids, other)
				}
			}
			deltas := make([]float64, len(ids))
			for j := range deltas {
				deltas[j] = []float64{-.08, -.02, 0, .02, .08}[rng.Intn(5)]
			}
			change := k.S2R6Intervention{Ordinals: ids, Delta: deltas, Clamp: (i+trial)%5 == 0}
			in := s2Input(row.Trial.Pre, pop)
			ctx, err := r.brain.S2R6Describe(in, change)
			s1Check(t, err)
			queryInput := s2r6UtilityInput(ctx, row.Trial.Pre.At, 4)
			start := time.Now()
			prediction := s2r6Read(t, r, queryInput)
			cost := time.Since(start).Nanoseconds()
			id := fmt.Sprintf("u-%d-%04d-%d", seed, i, trial)
			beforeDigest := k.NativeStateDigest(r.activation.CurrentState())
			// No intervention result, actual error or label is in this durable forecast.
			s1Check(t, json.NewEncoder(f).Encode(map[string]any{"ID": id, "OriginID": row.Trial.ID, "Input": queryInput, "InputSHA256": s2r3JSONHash(t, queryInput), "Prediction": prediction, "BeforeDigest": beforeDigest}))
			s1Check(t, f.Sync())
			before, after, err := r.brain.S2R6Counterfactual(in, change)
			s1Check(t, err)
			y := row.Trial.Error
			utility := ((before-y)*(before-y) - (after-y)*(after-y)) / 4
			if math.Abs(utility) > 1 {
				t.Fatal("invalid measured utility")
			}
			afterDigest := k.NativeStateDigest(r.activation.CurrentState())
			if beforeDigest != afterDigest {
				t.Fatal("detached intervention mutated activation")
			}
			out = append(out, s2r6UtilityRow{ID: id, OriginID: row.Trial.ID, Stream: row.Stream, Transfer: transfer, Intervention: change, Context: ctx, Input: queryInput, Before: before, After: after, Actual: y, Utility: utility, Prediction: prediction, PredictionNanoseconds: cost, BeforeDigest: beforeDigest, AfterDigest: afterDigest, ObservedAt: s2r4Now(r)})
		}
	}
	s1Write(t, filepath.Join(dir, "rows.json"), out)
	return out
}

func s2r6UtilityLesson(t *testing.T, r *BrainRuntime, id string, fit, check []s2r6UtilityRow, shuffle bool, seed int64) *CompositionLearningReceipt {
	t.Helper()
	e := k.CompositionExposure{ID: id, Source: "r6-measured-paired-causal-consequence", EpisodeID: id, IndependenceGroup: id, CausalGroup: "paired-native-numerical-interventions", Reliability: 1, Motor: []float64{0, 0, 0, 0}, After: s1Scene(s2r4Now(r))}
	e.Resolution = &k.NResolutionLesson{Proposal: &k.NMetricProposalConfig{Protocol: k.NMetricReuseV2, Strategy: "full-refit", PreserveShared: true, FixedBasis: true, FeatureLimit: 32}}
	for si, rows := range [][]s2r6UtilityRow{fit, check} {
		labels := make([]float64, len(rows))
		for i, row := range rows {
			labels[i] = row.Utility
		}
		if shuffle {
			rng := rand.New(rand.NewSource(seed + int64(si)))
			rng.Shuffle(len(labels), func(i, j int) { labels[i], labels[j] = labels[j], labels[i] })
		}
		for i, row := range rows {
			at := row.ObservedAt
			if at.IsZero() {
				at = row.Input.Scene.ObservedAt
			}
			o := k.NObservation{Input: row.Input, Outcome: labels[i], ObservedAt: at, EpisodeID: row.OriginID}
			if si == 0 {
				e.Resolution.Fit = append(e.Resolution.Fit, o)
			} else {
				e.Resolution.Check = append(e.Resolution.Check, o)
			}
		}
	}
	e.Input = e.Resolution.Fit[0].Input
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, s2r4Now(r))
	s1Check(t, err)
	if len(r.brain.BrainState.NResolution.Features) != 32 {
		t.Fatal("capacity growth")
	}
	return receipt
}

func s2r6Selected(rows []s2r6UtilityRow, start, stop int) []s2r6UtilityRow {
	// Two interventions for each original operation stay in the same split.
	return slices.Clone(rows[start:stop])
}

func TestS2R6Development(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	for _, seed := range []int{37, 83} {
		base := filepath.Join(work, fmt.Sprintf("development/seed-%d", seed))
		od := filepath.Join(base, "observer")
		r, w := s2r2Fork(t, cp, wal, od)
		rows := s2r4Collect(t, r, pop, od, "persistent", 171820000+int64(seed*100), 24, 16, false)
		s1Check(t, w.Close())
		bd := filepath.Join(base, "bootstrap")
		brain, bw := s2r2Fork(t, cp, wal, bd)
		s2r5ReplayObserver(t, brain, filepath.Join(od, "events.jsonl"))
		receipts := []*CompositionLearningReceipt{}
		for b := 0; b < 4; b++ {
			rec, _ := s2r5Lesson(t, brain, s2r5Arm{Name: "D32", Method: "readout", Capacity: 32, Refine: true, Blocks: true, Coverage: true}, fmt.Sprintf("r6-dev-bootstrap-%d-%d", seed, b), s2r4Subset(rows, 2*b, 2*b+2), s2r4Subset(rows, 8+2*b, 10+2*b), pop)
			receipts = append(receipts, rec)
		}
		s1Write(t, filepath.Join(bd, "receipts.json"), receipts)
		s2r3SaveCP(t, brain, filepath.Join(bd, "final.json"))
		s1Check(t, bw.Close())
		// Eight streams not used to build history become the contribution corpus.
		utility := s2r6CausalRows(t, brain, s2r4Subset(rows, 16, 24), pop, filepath.Join(base, "utility-acquisition"), 171830000+int64(seed), false)
		// Additional independent operations for developmental evaluation.
		eod := filepath.Join(base, "evaluation-observer")
		er, ew := s2r2Fork(t, cp, wal, eod)
		evalRows := s2r4Collect(t, er, pop, eod, "persistent", 171840000+int64(seed*100), 8, 16, false)
		s1Check(t, ew.Close())
		for _, arm := range []string{"C", "F"} {
			dir := filepath.Join(base, arm)
			br, ww := s2r2Fork(t, filepath.Join(bd, "final.json"), filepath.Join(bd, "events.jsonl"), dir)
			ur := []*CompositionLearningReceipt{}
			for b := 0; b < 4; b++ {
				// 32+32 independent original operations / block, 64+64 interventions.
				fit := s2r6Selected(utility, 0, 64)
				check := s2r6Selected(utility, 128, 192)
				if b%2 == 1 {
					fit = s2r6Selected(utility, 64, 128)
					check = s2r6Selected(utility, 192, 256)
				}
				ur = append(ur, s2r6UtilityLesson(t, br, fmt.Sprintf("r6-dev-%d-%s-%d", seed, arm, b), fit, check, arm == "F", int64(171850000+100*seed)))
				s2r3SaveCP(t, br, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
			}
			s1Write(t, filepath.Join(dir, "receipts.json"), ur)
			s2r3SaveCP(t, br, filepath.Join(dir, "final.json"))
			s1Check(t, ww.Close())
			for _, transfer := range []bool{false, true} {
				label := "seen"
				if transfer {
					label = "unseen"
				}
				s2r6CausalRows(t, br, evalRows, pop, filepath.Join(dir, "eval-"+label), 171860000+int64(seed), transfer)
			}
		}
		t.Log("development complete", seed)
	}
}

// Verifies that the new generic codec cannot obtain a target through a row.
func TestS2R6CodecCausalBoundary(t *testing.T) {
	ctx := k.S2R6Context{Measurements: make([]float64, 18), FeatureNodes: []k.NodeID{42}, ParameterDelta: []float64{.1}}
	a := s2r6UtilityInput(ctx, s1Base, 4)
	ctx.FeatureNodes = []k.NodeID{99999}
	ctx.ParameterDelta = []float64{7}
	b := s2r6UtilityInput(ctx, s1Base, 4)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("audit-only IDs/parameters leaked into codec")
	}
}

func s2r6Seed(t *testing.T) int {
	seed, err := strconv.Atoi(os.Getenv("HORIZON_S2R6_SEED"))
	s1Check(t, err)
	if !slices.Contains(s2r5Seeds, seed) {
		t.Fatal("unregistered seed")
	}
	return seed
}
