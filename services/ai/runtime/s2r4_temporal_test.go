//go:build horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Opt-in environment/acquisition adapter. No target or latent regime enters
// the input codec; every target follows a journaled actual neural operation.
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
	"strconv"
	"testing"
	"time"
)

type s2r4Row struct {
	Trial                             s2Trial
	Stream                            string
	Step                              int
	Condition                         string
	BeforeSequence, OperationSequence uint64
	PreviousID                        string
}

type s2r4Arm struct {
	Name                           string
	Capacity                       int
	Repair, NoHistory, Independent bool
}

var s2r4Arms = []s2r4Arm{
	{Name: "A32", Capacity: 32}, {Name: "B32", Capacity: 32, Repair: true},
	{Name: "C32", Capacity: 32, NoHistory: true}, {Name: "B0-32", Capacity: 32, Repair: true, NoHistory: true},
	{Name: "D32", Capacity: 32, Independent: true}, {Name: "D0-32", Capacity: 32, Independent: true, NoHistory: true},
	{Name: "A64", Capacity: 64}, {Name: "C64", Capacity: 64, NoHistory: true},
}

func s2r4Trials(rows []s2r4Row) []s2Trial {
	out := make([]s2Trial, len(rows))
	for i, r := range rows {
		out[i] = r.Trial
	}
	return out
}
func s2r4Lesson(t *testing.T, r *BrainRuntime, arm s2r4Arm, id string, fit, check []s2r4Row, pop [][]k.NodeID) (*CompositionLearningReceipt, map[string]any) {
	t.Helper()
	e := s2Lesson(1, s2r4Trials(fit), s2r4Trials(check), pop, s2r4Now(r))
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "s2r4-actual-sequential-neural-error"
	p := &k.NMetricProposalConfig{Protocol: k.NMetricAuditV1, Strategy: "combined", Fractions: append([]float64(nil), s2r2Fractions...), PreserveShared: true, FeatureLimit: arm.Capacity}
	if arm.Repair {
		p.Protocol = k.NMetricRefineV1
		p.RefineLeafDirections = true
	}
	e.Resolution.Proposal = p
	if arm.NoHistory {
		for _, set := range [][]k.NObservation{e.Resolution.Fit, e.Resolution.Check} {
			for i := range set {
				for j := 3; j < 6; j++ {
					set[i].Input.Sequence[1][j] = 0
				}
			}
		}
		e.Input = e.Resolution.Fit[0].Input
	}
	inputs := []k.CompositionInput{}
	ys := []float64{}
	ids := []string{}
	for _, o := range e.Resolution.Fit {
		inputs = append(inputs, o.Input)
		ys = append(ys, o.Outcome)
		ids = append(ids, o.EpisodeID)
	}
	audit, err := r.brain.S2R4FitAudit(inputs, ys, ids)
	s1Check(t, err)
	started := time.Now()
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, s2r4Now(r))
	s1Check(t, err)
	return receipt, map[string]any{"wall_nanoseconds": time.Since(started).Nanoseconds(), "fitting_audit": audit, "inventory": s2r3Inventory(t, r)}
}
func s2r4Subset(rows []s2r4Row, first, last int) []s2r4Row {
	streams := []string{}
	seen := map[string]bool{}
	for _, r := range rows {
		if !seen[r.Stream] {
			streams = append(streams, r.Stream)
			seen[r.Stream] = true
		}
	}
	include := map[string]bool{}
	for _, s := range streams[first:last] {
		include[s] = true
	}
	out := []s2r4Row{}
	for _, r := range rows {
		if include[r.Stream] {
			out = append(out, r)
		}
	}
	return out
}
func s2r4Retention(t *testing.T, source, candidate *BrainRuntime, rows []s2Trial, pop [][]k.NodeID) map[string]any {
	s1a := s1Predictions(t, source, s2S1Rows(rows), pop)
	s1b := s1Predictions(t, candidate, s2S1Rows(rows), pop)
	d := s1AuditMax(s1AuditValue(t, s1a), s1AuditValue(t, s1b))
	p := s2Protocol{LowCut: .005, HighCut: .02, Bins: []float64{.005, .01, .02, .04, .08}}
	a := s2MetricOf(rows, s2Predictions(t, source, rows, pop), p)
	b := s2MetricOf(rows, s2Predictions(t, candidate, rows, pop), p)
	witnessesExact := reflect.DeepEqual(source.brain.BrainState.NResolution.Witnesses, candidate.brain.BrainState.NResolution.Witnesses)
	policyExact := reflect.DeepEqual(source.brain.NativeControlSnapshot(), candidate.brain.NativeControlSnapshot())
	return map[string]any{"S1_max_delta": d, "S2_source": a, "S2_candidate": b, "S2_RMSE_limit": a.RMSE*1.05 + 1e-7, "witnesses_exact": witnessesExact, "native_policy_exact": policyExact, "pass": d <= 1e-7 && b.RMSE <= a.RMSE*1.05+1e-7 && witnessesExact && policyExact}
}
func TestS2R4DevelopmentLearning(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	source := orderedFixtureClone(t, cp)
	result := map[string]any{}
	for _, arm := range s2r4Arms {
		if arm.Independent || arm.Name == "C64" {
			continue
		}
		dir := filepath.Join(work, "development-learning-v2", arm.Name)
		r, w := s2r2Fork(t, cp, wal, dir)
		events, err := ReadEventLog(filepath.Join(work, "development/persistent/events.jsonl"))
		s1Check(t, err)
		for _, event := range events {
			if event.Sequence > 80735 {
				_, err := r.ApplyNativeControl(*event.NativeControl, event.Timestamp)
				s1Check(t, err)
			}
		}
		receipts := []*CompositionLearningReceipt{}
		progress := []any{}
		for b := 0; b < 2; b++ {
			fit := s2r4Subset(rows, 2*b, 2*b+2)
			check := s2r4Subset(rows, 8+2*b, 10+2*b)
			receipt, cost := s2r4Lesson(t, r, arm, fmt.Sprintf("s2r4-dev-%s-b%d", arm.Name, b), fit, check, pop)
			receipts = append(receipts, receipt)
			progress = append(progress, cost)
			s1Write(t, filepath.Join(dir, "receipts.json"), receipts)
			s1Write(t, filepath.Join(dir, "costs.json"), progress)
			s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
			t.Log(arm.Name, b, "commit", receipt.Diagnosis.Committed, receipt.Diagnosis.Cause, "features", len(r.brain.BrainState.NResolution.Features))
		}
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		auditRows := s2r4Subset(rows, 16, 24)
		pred := s2Predictions(t, r, s2r4Trials(auditRows), pop)
		noRows := s2r4Trials(auditRows)
		for i := range noRows {
			p := noRows[i].Pre
			p.History = []float64{0, 0, 0}
			noRows[i].Pre = p
		}
		s1Write(t, filepath.Join(dir, "predictions.json"), map[string]any{"full": pred, "no-history": s2Predictions(t, r, noRows, pop), "rows": auditRows})
		retention := s2r4Retention(t, source, r, old, pop)
		s1Write(t, filepath.Join(dir, "retention.json"), retention)
		result[arm.Name] = retention
		s1Check(t, w.Close())
	}
	s1Write(t, filepath.Join(work, "development-learning-summary.json"), result)
}

func TestS2R4Train(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R4_SEED"))
	s1Check(t, e)
	if seed != 37 && seed != 83 && seed != 151 {
		t.Fatal("unregistered seed")
	}
	if _, e := os.Stat(filepath.Join(work, "TRAINING_LOCK.json")); e != nil {
		t.Fatal("training protocol not frozen")
	}
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, fmt.Sprintf("training/seed-%d", seed))
	s1Check(t, os.MkdirAll(dir, 0700))
	observations := map[bool][]s2r4Row{}
	for _, independent := range []bool{false, true} {
		condition := "persistent"
		if independent {
			condition = "independent"
		}
		od := filepath.Join(dir, "observer-"+condition)
		r, w := s2r2Fork(t, cp, wal, od)
		observations[independent] = s2r4Collect(t, r, pop, od, condition, int64(2500000+1000*seed), 16, 16, false)
		s1Check(t, w.Close())
	}
	source := orderedFixtureClone(t, cp)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	for _, arm := range s2r4Arms {
		ad := filepath.Join(dir, arm.Name)
		r, w := s2r2Fork(t, cp, wal, ad)
		rows := observations[arm.Independent]
		// The same journaled observations are replayed into every corresponding
		// learner before numerical updates. Gains/latent regimes stay in WAL,
		// never in Fit or Check's encoded measurement vector.
		cond := "persistent"
		if arm.Independent {
			cond = "independent"
		}
		events, e := ReadEventLog(filepath.Join(dir, "observer-"+cond, "events.jsonl"))
		s1Check(t, e)
		for _, ev := range events {
			if ev.Sequence > 80735 {
				if ev.NativeControl == nil {
					t.Fatal("observer emitted non-native command")
				}
				_, e := r.ApplyNativeControl(*ev.NativeControl, ev.Timestamp)
				s1Check(t, e)
			}
		}
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		retentions := []any{}
		for b := 0; b < 4; b++ {
			fit := s2r4Subset(rows, 4*b, 4*b+2)
			check := s2r4Subset(rows, 4*b+2, 4*b+4)
			receipt, cost := s2r4Lesson(t, r, arm, fmt.Sprintf("s2r4-s%d-%s-b%d", seed, arm.Name, b), fit, check, pop)
			receipts = append(receipts, receipt)
			costs = append(costs, cost)
			retentions = append(retentions, s2r4Retention(t, source, r, old, pop))
			s1Write(t, filepath.Join(ad, "receipts.json"), receipts)
			s1Write(t, filepath.Join(ad, "costs.json"), costs)
			s1Write(t, filepath.Join(ad, "retention-progress.json"), retentions)
			s1Write(t, filepath.Join(ad, fmt.Sprintf("block-%d-fit.json", b)), fit)
			s1Write(t, filepath.Join(ad, fmt.Sprintf("block-%d-check.json", b)), check)
			s2r3SaveCP(t, r, filepath.Join(ad, fmt.Sprintf("block-%d.json", b)))
			t.Logf("seed %d arm %s block %d commit %v cause %s features %d", seed, arm.Name, b, receipt.Diagnosis.Committed, receipt.Diagnosis.Cause, len(r.brain.BrainState.NResolution.Features))
		}
		s2r3SaveCP(t, r, filepath.Join(ad, "final.json"))
		s1Check(t, w.Close())
	}
}

type s2r4Environment struct {
	ID            string
	Stream        string
	Step          int
	Gain          float64
	Regime        int
	Work          any
	Before, After map[k.NodeID]float64
}

func s2r4Now(r *BrainRuntime) time.Time {
	// Strictly ordered simulated event time. Nanosecond ticks keep unrelated
	// physical forgetting below the unchanged 1e-7 retention bound; wall-clock
	// inference/training cost is measured separately.
	return s1Base.Add(time.Duration(r.LastSequence()-80735+1) * time.Nanosecond)
}
func s2r4Command(t *testing.T, r *BrainRuntime, c NativeControlCommand) NativeControlResult {
	t.Helper()
	o, e := r.ApplyNativeControl(c, s2r4Now(r))
	s1Check(t, e)
	return o
}

func s2r4Paths(t *testing.T) (string, string, [][]k.NodeID) {
	t.Helper()
	root, work := os.Getenv("HORIZON_S2R4_SOURCE"), os.Getenv("HORIZON_S2R4_WORK")
	if root == "" || work == "" {
		t.Skip("explicit authenticated R2 source and isolated R4 work required")
	}
	if root == work || filepath.Dir(work) == root {
		t.Fatal("source/work overlap")
	}
	return root, work, s1Pop(t, filepath.Join(root, "source/source/source"))
}

func s2r4Collect(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, dir, condition string, seed int64, streams, steps int, shift bool) []s2r4Row {
	t.Helper()
	s1Check(t, os.MkdirAll(dir, 0700))
	ledger, e := os.OpenFile(filepath.Join(dir, "preoperation.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	defer ledger.Close()
	env, e := os.OpenFile(filepath.Join(dir, "environment.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	defer env.Close()
	out := []s2r4Row{}
	for stream := 0; stream < streams; stream++ {
		streamID := fmt.Sprintf("r4-%s-%d-stream-%02d", condition, seed, stream)
		u := rand.New(rand.NewSource(seed + int64(104729*stream)))
		latent := rand.New(rand.NewSource(seed + int64(104729*stream) + 30000001))
		var history, previous []float64
		historyAction, previousAction := -1, -1
		previousID := ""
		var after map[k.NodeID]float64
		regime := latent.Intn(2)
		for stepIndex := 0; stepIndex < steps; stepIndex++ {
			if condition == "independent" || latent.Float64() < .07 {
				regime = latent.Intn(2)
			}
			gain := .10 + .87*float64(regime) + .025*(latent.Float64()-.5)
			if shift {
				gain = .17 + .75*float64(regime) + .035*(latent.Float64()-.5)
			}
			action := 0
			if u.Float64() < .18 {
				action = 1
			}
			state := map[k.NodeID]float64{}
			for groupIndex, group := range pop {
				drive := .06 + .56*u.Float64()
				if shift {
					drive = .08 + .60*u.Float64()
				}
				// Sensory drive favors an existing upstream population and
				// leaves its downstream populations quiet. A gain change now
				// has a measurable effect on the actual recurrent transition.
				if groupIndex == 5 {
					drive = .40 + .22*u.Float64()
				}
				if groupIndex == 6 || groupIndex == 7 {
					drive = .02 + .04*u.Float64()
				}
				for _, id := range group {
					// Independent per-neuron drive prevents a common population
					// offset from exposing the preceding gain through exact spatial
					// contrasts. The complete 36-neuron state is audited offline.
					neuronDrive := math.Max(.001, math.Min(.9, drive+.08*(u.Float64()-.5)))
					state[id] = neuronDrive
					if after != nil {
						state[id] = .98*neuronDrive + .02*after[id]
					}
				}
			}
			id := fmt.Sprintf("%s-step-%02d", streamID, stepIndex)
			s2r4Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: state, Source: "s2r4-sequential-observed-stimulus", SampleID: id}}})
			p := s2Pre{Before: r.activation.CurrentState(), Previous: append([]float64(nil), previous...), Action: action, PreviousAction: previousAction, At: s2r4Now(r), History: append([]float64(nil), history...), HistoryAction: historyAction}
			sample := s1Sample{Before: p.Before, Previous: p.Previous, Action: p.Action, PreviousAction: p.PreviousAction, BeforeAt: p.At}
			p.Forecast = s1Predict(t, r, sample, pop)
			beforeRead := k.NativeStateDigest(r.activation.CurrentState())
			readSequence := r.LastSequence()
			estimate := s2Predict(t, r, p, pop)
			if beforeRead != k.NativeStateDigest(r.activation.CurrentState()) || readSequence != r.LastSequence() {
				t.Fatal("forecast/read mutated state")
			}
			encoded, e := json.Marshal(s2Input(p, pop))
			s1Check(t, e)
			beforeSequence := r.LastSequence()
			// This ledger deliberately contains no gain, regime, target, or poststate.
			s1Check(t, json.NewEncoder(ledger).Encode(map[string]any{"id": id, "stream": streamID, "step": stepIndex, "previous_id": previousID, "before_sequence": beforeSequence, "input": s2Input(p, pop), "input_sha256": s1Digest(encoded), "forecast": p.Forecast, "estimate": estimate}))
			s1Check(t, ledger.Sync())
			command := NativeControlCommand{ID: id + "-target", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}}
			if action == 0 {
				command.Step.EnvironmentGain = &gain
			}
			result := s2r4Command(t, r, command)
			after = result.State
			operationSequence := result.Sequence
			actual := s1Target(after, pop)
			residual := make([]float64, 3)
			history = make([]float64, 3)
			magnitude := 0.
			for q := range actual {
				residual[q] = actual[q] - p.Forecast[q]
				history[q] = math.Abs(residual[q])
				magnitude += history[q] / 3
			}
			s1Check(t, json.NewEncoder(env).Encode(s2r4Environment{ID: id, Stream: streamID, Step: stepIndex, Gain: gain, Regime: regime, Work: result.Work, Before: p.Before, After: after}))
			if !result.Stopped {
				stop := 1
				result = s2r4Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
			}
			s2r4Command(t, r, NativeControlCommand{ID: id + "-outcome", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: result.TraceDigest, Source: "s2r4-actual-neural-error", OutcomeID: id, Provenance: "observed-environment", Learn: false}})
			trial := s2Trial{ID: id, Pre: p, Actual: actual, Residual: residual, Error: magnitude, Estimate: estimate, AfterAt: p.At, InputSHA256: s1Digest(encoded)}
			out = append(out, s2r4Row{Trial: trial, Stream: streamID, Step: stepIndex, Condition: condition, BeforeSequence: beforeSequence, OperationSequence: operationSequence, PreviousID: previousID})
			previous = s1Previous(p.Before, pop)
			previousAction = action
			historyAction = action
			previousID = id
		}
	}
	s1Check(t, env.Sync())
	s1Write(t, filepath.Join(dir, "rows.json"), out)
	s2r3SaveCP(t, r, filepath.Join(dir, "observer-final.json"))
	return out
}

func TestS2R4Development(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	if _, e := os.Stat(work); !os.IsNotExist(e) {
		t.Fatal("work must be new")
	}
	s1Check(t, os.MkdirAll(work, 0700))
	cp, wal := s2r3Source(root)
	snap := s1Snapshot(t, cp)
	sum, e := checkpointChecksumForSchema(snap)
	s1Check(t, e)
	if snap.Sequence != 80735 || snap.Checksum != s2r3Checksum || sum != s2r3Checksum {
		t.Fatal("BLOCKED: source identity/checksum")
	}
	s2r2Copy(t, cp, filepath.Join(work, "00-source.json"))
	s2r2Copy(t, wal, filepath.Join(work, "source-events.jsonl"))
	for _, condition := range []string{"independent", "persistent"} {
		dir := filepath.Join(work, "development", condition)
		r, w := s2r2Fork(t, cp, wal, dir)
		if r.LastSequence() != 80735 {
			t.Fatal("source WAL advanced")
		}
		if r.activation.Memory != r.brain || r.learning.Kb != r.brain || r.dnf.Brain() != r.brain {
			t.Fatal("split Brain owner")
		}
		inv := s2r3Inventory(t, r)
		s1Write(t, filepath.Join(work, "SOURCE_NATIVE_VERIFICATION.json"), map[string]any{"sequence": snap.Sequence, "native_checksum": sum, "inventory": inv, "checkpoint_sha256": s2r3FileHash(t, cp), "journal_sha256": s2r3FileHash(t, wal), "source_recovered_exact": bytes.Equal(continuationBytes(t, r), continuationBytes(t, orderedFixtureClone(t, cp)))})
		rows := s2r4Collect(t, r, pop, dir, condition, 2400011, 24, 32, false)
		inputs := []k.CompositionInput{}
		zero := []k.CompositionInput{}
		for _, row := range rows {
			inputs = append(inputs, s2Input(row.Trial.Pre, pop))
			p := row.Trial.Pre
			p.History = []float64{0, 0, 0}
			zero = append(zero, s2Input(p, pop))
		}
		audit, e := r.brain.S2R3Inspect(inputs, s2r3Old(t, cp))
		s1Check(t, e)
		s1Write(t, filepath.Join(dir, "representation.json"), audit)
		audit, e = r.brain.S2R3Inspect(zero, s2r3Old(t, cp))
		s1Check(t, e)
		s1Write(t, filepath.Join(dir, "representation-no-history.json"), audit)
		s1Check(t, w.Close())
		t.Log("acquired native", condition, len(rows))
	}
}
