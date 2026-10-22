package knowledge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func memoryTimeFixture(t *testing.T, retention float64) (*Brain, *Synapse, time.Time) {
	t.Helper()
	at := time.Date(2026, 10, 20, 0, 0, 0, 123, time.UTC)
	b := NewBrain()
	a, z := b.Store("a"), b.Store("z")
	b.ConnectAt(a, z, .8, .8, false, at)
	s := a.Synapses[z.ID][0]
	s.Dynamic.Eligibility = .9
	b.BrainState.MemoryState["retention"] = retention
	b.BrainState.ExperienceTraces = []ExperienceTrace{{CurrentAt: at, MemoryPriority: .6}}
	return b, s, at
}

func TestPhysicalMemoryTimeComposesWithoutAdditionalExperience(t *testing.T) {
	for _, retention := range []float64{.2, .6, 1} {
		for _, elapsed := range []time.Duration{time.Hour, 24 * time.Hour, 72 * time.Hour} {
			for _, steps := range []int{1, 24, 96} {
				t.Run(fmt.Sprintf("retention%.1f/%s/%d", retention, elapsed, steps), func(t *testing.T) {
					b, s, at := memoryTimeFixture(t, retention)
					for i := 1; i <= steps; i++ {
						b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Duration(i)*elapsed/time.Duration(steps)), PhysicalMemoryDynamics)
					}
					factor := math.Exp(-elapsed.Hours() * (1 - retention) / 24)
					for name, values := range map[string][2]float64{
						"weight":     {s.Dynamic.Weight, .8 * factor},
						"confidence": {s.Dynamic.Confidence, .8 * math.Sqrt(factor)},
						"priority":   {b.BrainState.ExperienceTraces[0].MemoryPriority, .6 * factor},
					} {
						if math.Abs(values[0]-values[1]) > 1e-12 {
							t.Fatalf("%s got %.15f want %.15f", name, values[0], values[1])
						}
					}
					if s.Weight != s.Dynamic.Weight || s.Confidence != s.Dynamic.Confidence || s.Dynamic.Eligibility != .9 || s.Dynamic.Frequency != 1 || !s.Dynamic.LastModification.Equal(at) || !s.Dynamic.LastEligibilityUpdate.Equal(at) {
						t.Fatal("passive retention changed learning/eligibility clocks or desynchronized compatibility state")
					}
					before, _ := b.CanonicalJSON()
					for i := 0; i < 100; i++ {
						b.ApplyMemoryDynamicsWithEncoding(at.Add(elapsed), PhysicalMemoryDynamics)
					}
					b.ApplyMemoryDynamicsWithEncoding(at.Add(-time.Hour), PhysicalMemoryDynamics)
					after, _ := b.CanonicalJSON()
					if !bytes.Equal(before, after) {
						t.Fatal("duplicate or backwards time changed memory")
					}
				})
			}
		}
	}
}

func TestPhysicalMemoryRestartKeepsNanosecondClockAndNextDecay(t *testing.T) {
	b, _, at := memoryTimeFixture(t, .6)
	b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Hour+789*time.Nanosecond), PhysicalMemoryDynamics)
	data, err := b.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "brain.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	restored := NewBrain()
	if err = restored.Load(path); err != nil {
		t.Fatal(err)
	}
	copy, _ := restored.CanonicalJSON()
	if !bytes.Equal(data, copy) || !restored.BrainState.MemoryDecay.AppliedAt.Equal(at.Add(time.Hour+789*time.Nanosecond)) {
		t.Fatal("restart lost exact decay clock or state")
	}
	for _, brain := range []*Brain{b, restored} {
		brain.ApplyMemoryDynamicsWithEncoding(at.Add(2*time.Hour), PhysicalMemoryDynamics)
	}
	want, _ := b.CanonicalJSON()
	got, _ := restored.CanonicalJSON()
	if !bytes.Equal(want, got) {
		t.Fatal("next passive transition diverged after restart")
	}
}

func TestPhysicalMemoryNewLinksAndTracesStartAtCreation(t *testing.T) {
	b, old, at := memoryTimeFixture(t, .6)
	b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Hour), PhysicalMemoryDynamics)
	a, z := b.Store("fresh"), b.Store("target")
	b.ConnectAt(a, z, .8, .8, false, at.Add(90*time.Minute))
	b.BrainState.ExperienceTraces = append(b.BrainState.ExperienceTraces, ExperienceTrace{CurrentAt: at.Add(90 * time.Minute), MemoryPriority: .6})
	b.ApplyMemoryDynamicsWithEncoding(at.Add(2*time.Hour), PhysicalMemoryDynamics)
	if math.Abs(old.Dynamic.Weight-.8*math.Exp(-2*.4/24)) > 1e-12 || math.Abs(a.Synapses[z.ID][0].Dynamic.Weight-.8*math.Exp(-.5*.4/24)) > 1e-12 || math.Abs(b.BrainState.ExperienceTraces[1].MemoryPriority-.6*math.Exp(-.5*.4/24)) > 1e-12 {
		t.Fatal("creation time was charged as prior memory age")
	}
}

func TestPhysicalMemoryAdoptionDoesNotChargeLegacyAgeAgain(t *testing.T) {
	b, s, at := memoryTimeFixture(t, .6)
	b.ApplyMemoryDynamics(at.Add(time.Hour))
	weight, confidence, eligibility := s.Dynamic.Weight, s.Dynamic.Confidence, s.Dynamic.Eligibility
	b.ApplyMemoryDynamicsWithEncoding(at.Add(2*time.Hour), PhysicalMemoryDynamics)
	if math.Abs(s.Dynamic.Weight-weight*math.Exp(-.4/24)) > 1e-12 || math.Abs(s.Dynamic.Confidence-confidence*math.Exp(-.2/24)) > 1e-12 || s.Dynamic.Eligibility != eligibility {
		t.Fatal("adoption charged elapsed legacy time or consumed eligibility")
	}
}

func TestPhysicalMemoryPolicyChangesAffectOnlySubsequentIntervals(t *testing.T) {
	b, s, at := memoryTimeFixture(t, .2)
	b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Hour), PhysicalMemoryDynamics)
	b.BrainState.MemoryState["retention"] = .9
	b.ApplyMemoryDynamicsWithEncoding(at.Add(2*time.Hour), PhysicalMemoryDynamics)
	if math.Abs(s.Dynamic.Weight-.8*math.Exp(-(.8+.1)/24)) > 1e-12 {
		t.Fatal("policy change retroactively changed prior interval")
	}
}

func TestPhysicalMemoryClockBoundariesAndUnknownCreation(t *testing.T) {
	for _, at := range []time.Time{time.Unix(-3600, 0).UTC(), time.Unix(0, 0).UTC(), time.Unix(3600, 0).UTC()} {
		b, s, _ := memoryTimeFixture(t, .6)
		s.Dynamic.LastModification = at.Add(-time.Hour)
		b.ApplyMemoryDynamics(at)
		before := s.Dynamic.Weight
		b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Hour), PhysicalMemoryDynamics)
		if math.Abs(s.Dynamic.Weight-before*math.Exp(-.4/24)) > 1e-12 {
			t.Fatal("legacy adoption lost pre-/at-epoch time")
		}
	}
	b, s, at := memoryTimeFixture(t, .6)
	s.Dynamic.LastModification = at.Add(2 * time.Hour)
	b.BrainState.ExperienceTraces[0].CurrentAt = at.Add(2 * time.Hour)
	b.ApplyMemoryDynamicsWithEncoding(at.Add(time.Hour), PhysicalMemoryDynamics)
	if s.Weight != .8 || b.BrainState.ExperienceTraces[0].MemoryPriority != .6 {
		t.Fatal("future memory decayed before creation")
	}
	b.ApplyMemoryDynamicsWithEncoding(at.Add(3*time.Hour), PhysicalMemoryDynamics)
	if math.Abs(s.Weight-.8*math.Exp(-.4/24)) > 1e-12 {
		t.Fatal("future creation did not establish its own origin")
	}
	s.Dynamic.LastModification = time.Time{}
	b.BrainState.ExperienceTraces[0].CurrentAt = time.Time{}
	weight, priority := s.Weight, b.BrainState.ExperienceTraces[0].MemoryPriority
	b.ApplyMemoryDynamicsWithEncoding(at.Add(4*time.Hour), PhysicalMemoryDynamics)
	if s.Weight != weight || b.BrainState.ExperienceTraces[0].MemoryPriority != priority {
		t.Fatal("missing creation time was invented")
	}
}

func TestPhysicalMemoryCanonicalVersionGate(t *testing.T) {
	b, _, at := memoryTimeFixture(t, .6)
	b.ApplyMemoryDynamicsWithEncoding(at, PhysicalMemoryDynamics)
	data, _ := b.CanonicalJSON()
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["schema_version"] != float64(3) {
		t.Fatal("physical clock not versioned")
	}
	for _, mutate := range []func(map[string]any){
		func(r map[string]any) { r["schema_version"] = 2 },
		func(r map[string]any) { r["schema_version"] = 4 },
		func(r map[string]any) { delete(r, "memory_decay") },
		func(r map[string]any) { r["memory_decay"].(map[string]any)["applied_at"] = "0001-01-01T00:00:00Z" },
		func(r map[string]any) { r["memory_decay"].(map[string]any)["encoding"] = "future" },
	} {
		var changed map[string]any
		_ = json.Unmarshal(data, &changed)
		mutate(changed)
		packet, _ := json.Marshal(changed)
		if ValidateMemoryDecayState(packet, true) == nil {
			t.Fatal("invalid memory state accepted")
		}
	}
}
