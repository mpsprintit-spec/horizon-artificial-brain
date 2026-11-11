package runtime

// This is an isolated, numeric-only laboratory test. It starts with NewBrain(),
// uses the existing BrainRuntime and its canonical synapses/patterns, and never
// imports the user's persistent memory, the production CLI, or an external model.
// The labels in this test describe the teacher's measurement protocol only.
import (
    "encoding/json"
    "fmt"
    "math"
    "math/rand"
    "os"
    "path/filepath"
    "slices"
    "testing"
    "time"

    "github.com/project-horizon/horizon-core/services/ai/knowledge"
    "github.com/project-horizon/horizon-core/services/ai/learning"
)

func relationLabPopulationEquals(a, b []knowledge.NodeID) bool {
    if len(a) != len(b) {
        return false
    }
    x, y := slices.Clone(a), slices.Clone(b)
    slices.Sort(x)
    slices.Sort(y)
    return slices.Equal(x, y)
}

func relationLabScore(t *testing.T, path string, populations [][]knowledge.NodeID, context int, at time.Time, ablate bool) (float64, knowledge.PredictionAssessment) {
    t.Helper()
    probe := orderedFixtureClone(t, path)
    if ablate {
        // The ablation removes the acquired conditional-pattern channel.
        // It does not change the teacher or encode the expected answer.
        probe.brain.Patterns = knowledge.NewPatternIndex()
    }
    trace := &knowledge.ObservationTrace{
        Stimulus: [][]knowledge.NodeID{populations[0], populations[1], populations[2]},
        Context:  [][]knowledge.NodeID{populations[context]},
    }
    out, err := probe.CognitiveProcess(Event{
        ID: fmt.Sprintf("relational-probe-%d", context),
        EpisodeID: "heldout-relation",
        Timestamp: at,
        NeuralTrace: trace,
        Cycles: 1,
    })
    if err != nil {
        t.Fatal(err)
    }
    prediction := probe.activation.PredictionSnapshot().State
    mean := func(pop []knowledge.NodeID) float64 {
        var sum float64
        for _, id := range pop {
            sum += prediction[id]
        }
        return sum / float64(len(pop))
    }
    assessment := knowledge.PredictionAssessment{}
    if out.PredictionAssessment != nil {
        assessment = *out.PredictionAssessment
    }
    return mean(populations[6]) - mean(populations[7]), assessment
}

func TestRelationalContradictionEmptyBrain(t *testing.T) {
    const lessonsPerArm = 10
    for _, seed := range []int64{7, 17, 47} {
        t.Run(fmt.Sprintf("numeric-seed-%d", seed), func(t *testing.T) {
            at := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
            brain := knowledge.NewBrain()
            if len(brain.Patterns.All()) != 0 || len(brain.SurfaceAnnotationsSnapshot()) != 0 {
                t.Fatal("experiment must start with an empty learned brain")
            }
            r := NewBrainRuntime(brain)
            if r == nil {
                t.Fatal("could not bind an empty canonical Brain")
            }

            // Opaque populations: [subject, relation, object,
            // context-A, context-B, contested-context, outcome-1, outcome-2].
            // Axes and allocation order vary across seeds. No word/semantic
            // representation enters the Brain.
            permutation := rand.New(rand.NewSource(seed)).Perm(8)
            populations := make([][]knowledge.NodeID, 8)
            for i := range populations {
                values := make([]float64, 8)
                values[permutation[i]] = 1
                grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector(values), at)
                if err != nil {
                    t.Fatal(err)
                }
                if len(grounded.Population) == 0 {
                    t.Fatal("numeric grounding yielded an empty population")
                }
                populations[i] = slices.Clone(grounded.Population)
                for j := 0; j < i; j++ {
                    if relationLabPopulationEquals(populations[i], populations[j]) {
                        t.Fatal("different relational coordinates collapsed into one population")
                    }
                }
            }
            if len(brain.SurfaceAnnotationsSnapshot()) != 0 {
                t.Fatal("numeric grounding created surface-language annotations")
            }
            foundation := orderedSave(t, r)
            probeAt := at.Add(2 * time.Second)
            baselineA, _ := relationLabScore(t, foundation, populations, 3, probeAt, false)
            baselineB, _ := relationLabScore(t, foundation, populations, 4, probeAt, false)

            type evidenceArm struct {
                context, outcome int
                source, independent, cause, contradiction string
            }
            arms := []evidenceArm{
                {3, 6, "observer-a", "instrument-a", "cause-a", ""},
                {4, 7, "observer-b", "instrument-b", "cause-b", ""},
                {5, 6, "observer-c", "instrument-c", "cause-c", "same-edge-conflict"},
                {5, 7, "observer-d", "instrument-d", "cause-d", "same-edge-conflict"},
            }
            train := func(flip bool) (string, *BrainRuntime) {
                learner := orderedFixtureClone(t, foundation)
                order := rand.New(rand.NewSource(seed + 101))
                for epoch := 0; epoch < lessonsPerArm; epoch++ {
                    indices := order.Perm(len(arms))
                    for _, index := range indices {
                        arm := arms[index]
                        outcome := arm.outcome
                        if flip {
                            outcome = 13 - outcome // flip only the teacher's 6/7 outcome populations
                        }
                        e := learning.Experience{
                            NeuralSequence: [][]knowledge.NodeID{
                                populations[0], populations[1], populations[2], populations[outcome],
                            },
                            NeuralContext: [][]knowledge.NodeID{populations[arm.context]},
                            Weight: .9, Confidence: .9, Reliability: .8,
                            Source: arm.source, Modality: "internal-numeric",
                            IndependenceGroup: arm.independent, CausalGroup: arm.cause,
                            ContradictionSet: arm.contradiction,
                            ExperienceID: fmt.Sprintf("%d-%d-%d", seed, epoch, index),
                            EpisodeID: fmt.Sprintf("measurement-%d-%d", epoch, index),
                            Timestamp: at.Add(time.Second),
                        }
                        if _, err := learner.LearnExperience(e, at.Add(time.Second)); err != nil {
                            t.Fatal(err)
                        }
                    }
                }
                if len(learner.brain.SurfaceAnnotationsSnapshot()) != 0 {
                    t.Fatal("relational lessons created lexical or semantic labels")
                }
                return orderedSave(t, learner), learner
            }

            trainedPath, trainedBrain := train(false)
            trainedA, _ := relationLabScore(t, trainedPath, populations, 3, probeAt, false)
            trainedB, _ := relationLabScore(t, trainedPath, populations, 4, probeAt, false)
            contested, assessment := relationLabScore(t, trainedPath, populations, 5, probeAt, false)
            ablatedA, _ := relationLabScore(t, trainedPath, populations, 3, probeAt, true)
            ablatedB, _ := relationLabScore(t, trainedPath, populations, 4, probeAt, true)
            flippedPath, _ := train(true)
            flippedA, _ := relationLabScore(t, flippedPath, populations, 3, probeAt, false)
            flippedB, _ := relationLabScore(t, flippedPath, populations, 4, probeAt, false)

            // The two contradictory reports about the SAME numeric tuple under
            // the SAME context must remain as separate learned outcome paths.
            // Repeating a single source does NOT create ten independent sources.
            found := map[int]bool{}
            for _, pattern := range trainedBrain.brain.Patterns.All() {
                if pattern.Observation == nil || len(pattern.Observation.Context) != 1 ||
                    !relationLabPopulationEquals(pattern.Observation.Context[0], populations[5]) {
                    continue
                }
                side := -1
                if slices.Contains(populations[6], pattern.Result) {
                    side = 6
                }
                if slices.Contains(populations[7], pattern.Result) {
                    side = 7
                }
                if side < 0 {
                    continue
                }
                if pattern.Frequency != lessonsPerArm ||
                    knowledge.CountEvidenceProvenance(pattern.Evidence).IndependentGroups != 1 {
                    t.Fatalf("lost repeated-but-not-independent evidence: side=%d freq=%d provenance=%+v",
                        side, pattern.Frequency, knowledge.CountEvidenceProvenance(pattern.Evidence))
                }
                for _, e := range pattern.Evidence {
                    if e.ContradictionSet != "same-edge-conflict" {
                        t.Fatal("conflicting evidence lost its contradiction-set provenance")
                    }
                }
                found[side] = true
            }
            if !found[6] || !found[7] {
                t.Fatal("one of the mutually contradictory outcomes was overwritten")
            }
            t.Logf(
                "seed=%d blank=(%.4f,%.4f) learned=(%.4f,%.4f) conflict_margin=%.4f "+
                    "conflict_ambiguity=%.4f ablation=(%.4f,%.4f) reversed=(%.4f,%.4f) patterns=%d",
                seed, baselineA, baselineB, trainedA, trainedB, contested,
                assessment.Ambiguity, ablatedA, ablatedB, flippedA, flippedB,
                len(trainedBrain.brain.Patterns.All()),
            )
            if trainedA < .07 || trainedB > -.07 {
                t.Fatalf("neural context did not disambiguate conflicting relational evidence: A=%f B=%f", trainedA, trainedB)
            }
            if flippedA > -.07 || flippedB < .07 {
                t.Fatalf("prediction did not follow reversed teacher outcomes: A=%f B=%f", flippedA, flippedB)
            }
            benefit := trainedA - trainedB
            if benefit <= (baselineA-baselineB)+.10 || benefit <= (ablatedA-ablatedB)+.10 {
                t.Fatalf("relational advantage predates teaching or survives pattern ablation: blank=%f learned=%f ablated=%f",
                    baselineA-baselineB, benefit, ablatedA-ablatedB)
            }
            if math.IsNaN(contested) || math.IsInf(contested, 0) {
                t.Fatal("contradictory case produced non-finite prediction")
            }
            // One reproducible trained Brain checkpoint is exported only when
            // the isolated CI workflow supplies an explicit artifact directory.
            // Nothing is written to production memory or repository main.
            if dir := os.Getenv("HORIZON_RELATIONAL_LAB_ARTIFACT_DIR"); seed == 17 && dir != "" {
                if err := os.MkdirAll(dir, 0700); err != nil {
                    t.Fatal(err)
                }
                checkpoint, err := os.ReadFile(trainedPath)
                if err != nil {
                    t.Fatal(err)
                }
                if err := os.WriteFile(filepath.Join(dir, "trained-brain-seed17.json"), checkpoint, 0600); err != nil {
                    t.Fatal(err)
                }
                report := struct {
                    Experiment string `json:"experiment"`
                    Seed int64 `json:"seed"`
                    Origin string `json:"origin"`
                    PopulationIDs [][]knowledge.NodeID `json:"population_ids"`
                    Lessons int `json:"lessons"`
                    PatternCount int `json:"pattern_count"`
                    BaselineMargins [2]float64 `json:"baseline_margins"`
                    LearnedMargins [2]float64 `json:"learned_margins"`
                    ContestedMargin float64 `json:"contested_margin"`
                    ContestedAmbiguity float64 `json:"contested_ambiguity"`
                    PatternAblation [2]float64 `json:"pattern_ablation_margins"`
                    ReversedTeacher [2]float64 `json:"reversed_teacher_margins"`
                    Limitations string `json:"limitations"`
                }{
                    Experiment: "numeric-conditional-relational-contradiction-v1",
                    Seed: seed, Origin: "knowledge.NewBrain(), no imported knowledge",
                    PopulationIDs: populations, Lessons: lessonsPerArm * len(arms),
                    PatternCount: len(trainedBrain.brain.Patterns.All()),
                    BaselineMargins: [2]float64{baselineA, baselineB},
                    LearnedMargins: [2]float64{trainedA, trainedB},
                    ContestedMargin: contested, ContestedAmbiguity: assessment.Ambiguity,
                    PatternAblation: [2]float64{ablatedA, ablatedB},
                    ReversedTeacher: [2]float64{flippedA, flippedB},
                    Limitations: "Teacher-supplied numeric factors; no demonstrated autonomous relation discovery, semantics, or real-world transfer.",
                }
                data, err := json.MarshalIndent(report, "", "  ")
                if err != nil {
                    t.Fatal(err)
                }
                if err := os.WriteFile(filepath.Join(dir, "metrics-seed17.json"), append(data, '
'), 0600); err != nil {
                    t.Fatal(err)
                }
                t.Logf("exported isolated trained Brain checkpoint (%d bytes) to artifact directory", len(checkpoint))
            }
        })
    }
}
