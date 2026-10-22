package knowledge

import (
	"bytes"
	"math/rand"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func spellingSource(t *testing.T, k *KnowledgeBase, seed int64, at time.Time) []NodeID {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	values := make([]float64, 16)
	for i := range values {
		values[i] = rng.Float64()*2 - 1
	}
	p, err := k.ProjectVectorPopulationAt(NewNeuralVector(values), .999999, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	var ids []NodeID
	for _, unit := range p.Units {
		ids = append(ids, unit.NodeID)
	}
	return ids
}

func teachSpelling(t *testing.T, k *KnowledgeBase, ids []NodeID, text string, count int, at time.Time) {
	t.Helper()
	e, err := NewSpellingExposure(ids, text, .8, .8)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		if err := k.LearnSpellingAt(e, at); err != nil {
			t.Fatal(err)
		}
	}
}

func assertSpelling(t *testing.T, k *KnowledgeBase, ids []NodeID, text string) SpellingReadout {
	t.Helper()
	out, err := k.ReadLearnedSpelling(ids)
	if err != nil || out.Status != SpellingDecoded || out.Text != text {
		t.Fatalf("wanted %q; got %+v: %v", text, out, err)
	}
	return out
}

func TestSynapticSpellingUnicodeLabelsDriftPersistenceAndAblation(t *testing.T) {
	words := []string{"lantai", "bola", "Núra🌌\x00", "e\u0301", "é", ""}
	for allocation, seed := range []int64{7, 17, 47} {
		k := NewBrain()
		at := time.Unix(100, 0).UTC()
		for i := 0; i < allocation; i++ {
			spellingSource(t, k, seed*1000+int64(i), at)
		}
		var populations [][]NodeID
		for i, word := range words {
			ids := spellingSource(t, k, seed*100+int64(i), at)
			populations = append(populations, ids)
			out, err := k.ReadLearnedSpelling(ids)
			if err != nil || out.Status != SpellingUntrained || out.Text != "" {
				t.Fatal("untrained population fabricated a spelling")
			}
			teachSpelling(t, k, ids, word, 4, at)
		}
		// All spellings use one shared positional bit bank, not per-word units.
		bank, _, err := k.spellingBankLocked()
		if err != nil || len(bank) != 6*22+1 {
			t.Fatalf("output bank duplicated words/characters: slots=%d err=%v", len(bank), err)
		}
		beforeUnits := len(k.Registry.Nodes())
		for mode := 0; mode < 3; mode++ {
			var annotations []SurfaceAnnotation
			if mode != 1 {
				for i, ids := range populations {
					annotations = append(annotations, SurfaceAnnotation{Surface: []string{"real-label", "forged-label"}[mode/2], Modality: "language", NodeID: ids[0], Population: ids, LastSeen: at.Add(time.Duration(i) * time.Second)})
				}
			}
			if err := k.RestoreSurfaceAnnotations(annotations); err != nil {
				t.Fatal(err)
			}
			// Member-vector drift cannot redirect the learned output weights.
			for _, node := range k.Registry.Nodes() {
				for i := range node.Representation {
					node.Representation[i] = .9*node.Representation[i] + .05
				}
			}
			canonical, err := k.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			for i, ids := range populations {
				out := assertSpelling(t, k, ids, words[i])
				reversed := slices.Clone(ids)
				slices.Reverse(reversed)
				if !reflect.DeepEqual(out, assertSpelling(t, k, reversed, words[i])) {
					t.Fatal("source ID order changed numeric readout")
				}
			}
			after, err := k.CanonicalJSON()
			if err != nil || !bytes.Equal(canonical, after) || len(k.Registry.Nodes()) != beforeUnits {
				t.Fatal("decoding changed the Brain or recruited units")
			}
			if bytes.Contains(after, []byte("lantai")) || bytes.Contains(after, []byte("forged-label")) {
				t.Fatal("canonical learned spelling depends on literal words")
			}
		}
		path := filepath.Join(t.TempDir(), "brain.json")
		if err := k.Save(path); err != nil {
			t.Fatal(err)
		}
		restored := NewBrain()
		if err := restored.Load(path); err != nil {
			t.Fatal(err)
		}
		for i, ids := range populations {
			assertSpelling(t, restored, ids, words[i])
			for _, id := range ids {
				restored.Registry.GetByID(id).Synapses = nil
			}
			out, err := restored.ReadLearnedSpelling(ids)
			if err != nil || out.Status != SpellingUntrained || out.Text != "" {
				t.Fatal("spelling survived synapse ablation with labels/coordinates retained")
			}
		}
	}
}

func TestSynapticSpellingAmbiguityMissingSupportAndNoPartialText(t *testing.T) {
	k, at := NewBrain(), time.Unix(100, 0).UTC()
	a, b, unknown := spellingSource(t, k, 1, at), spellingSource(t, k, 2, at), spellingSource(t, k, 3, at)
	teachSpelling(t, k, a, "A", 1, at)
	teachSpelling(t, k, b, "B", 1, at)
	for _, test := range []struct {
		population []NodeID
		status     SpellingStatus
	}{
		{append(slices.Clone(a), b...), SpellingAmbiguous},
		{append(slices.Clone(a), unknown...), SpellingIncomplete},
		{unknown, SpellingUntrained},
	} {
		out, err := k.ReadLearnedSpelling(test.population)
		if err != nil || out.Status != test.status || out.Text != "" || len(out.Scalars) != 0 {
			t.Fatalf("uncertain population produced text: %+v, %v", out, err)
		}
	}
	bank, _, err := k.spellingBankLocked()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range a {
		delete(k.Registry.GetByID(id).Synapses, bank[spellingCoordinate{1, spellingEndBit}])
	}
	out, err := k.ReadLearnedSpelling(a)
	if err != nil || out.Position != 1 || out.Status != SpellingUntrained || out.Text != "" || len(out.Scalars) != 0 {
		t.Fatal("missing learned termination returned a partial spelling")
	}
}

func TestSynapticSpellingBitWeightsAreCausalAndCorrectionRetainsHistory(t *testing.T) {
	k, at := NewBrain(), time.Unix(100, 0).UTC()
	ids := spellingSource(t, k, 11, at)
	teachSpelling(t, k, ids, "B", 4, at)
	bank, _, err := k.spellingBankLocked()
	if err != nil {
		t.Fatal(err)
	}
	target := bank[spellingCoordinate{0, 0}]
	for _, id := range ids {
		for _, s := range k.Registry.GetByID(id).Synapses[target] {
			s.Inhibitory = !s.Inhibitory // isolated causal intervention: B -> C
		}
	}
	assertSpelling(t, k, ids, "C")
	teachSpelling(t, k, ids, "B", 8, at.Add(time.Second))
	assertSpelling(t, k, ids, "B")
	for _, id := range ids {
		list := k.Registry.GetByID(id).Synapses[target]
		if list.FindDynamic(false) == nil || list.FindDynamic(true) == nil || list.FindDynamic(false).Dynamic.Frequency == 0 || list.FindDynamic(true).Dynamic.Frequency == 0 {
			t.Fatal("corrective supervision deleted the prior connection/history")
		}
	}
}

func TestSynapticSpellingLimitsAndScalarValidation(t *testing.T) {
	k, at := NewBrain(), time.Unix(100, 0).UTC()
	ids := spellingSource(t, k, 19, at)
	text := strings.Repeat("🌌", MaxSpellingScalars)
	teachSpelling(t, k, ids, text, 1, at)
	assertSpelling(t, k, ids, text)
	for _, invalid := range []string{text + "A", string([]byte{0xff})} {
		if _, err := NewSpellingExposure(ids, invalid, .8, .8); err == nil {
			t.Fatal("invalid/over-budget UTF-8 accepted")
		}
	}
	for _, scalar := range []uint32{0xd800, 0xdfff, 0x110000} {
		e := SpellingExposure{Codec: SpellingCodecV1, Population: ids, Scalars: []uint32{scalar}, Weight: .8, Confidence: .8}
		if err := k.LearnSpellingAt(e, at); err == nil {
			t.Fatal("non-scalar Unicode value accepted")
		}
	}
}

func TestSynapticSpellingRejectsCorruptBankAndInvalidDecodedScalar(t *testing.T) {
	k, at := NewBrain(), time.Unix(100, 0).UTC()
	ids := spellingSource(t, k, 31, at)
	teachSpelling(t, k, ids, "\x00", 1, at)
	bank, _, err := k.spellingBankLocked()
	if err != nil {
		t.Fatal(err)
	}
	// A controlled weight intervention reconstructs a surrogate, which must
	// be rejected instead of silently converted to Unicode replacement text.
	for bit := 0; bit < spellingScalarBits; bit++ {
		if 0xd800&(1<<uint(bit)) == 0 {
			continue
		}
		for _, id := range ids {
			for _, s := range k.Registry.GetByID(id).Synapses[bank[spellingCoordinate{0, bit}]] {
				s.Inhibitory = false
			}
		}
	}
	out, err := k.ReadLearnedSpelling(ids)
	if err != nil || out.Status != SpellingInvalid || out.Text != "" || len(out.Scalars) != 0 {
		t.Fatal("invalid learned scalar produced text")
	}
	for _, corruption := range []string{"duplicate", "nonintegral", "missing-unit"} {
		copyBrain := NewBrain()
		path := filepath.Join(t.TempDir(), "copy.json")
		if err := k.Save(path); err != nil {
			t.Fatal(err)
		}
		if err := copyBrain.Load(path); err != nil {
			t.Fatal(err)
		}
		var index int
		for i, p := range copyBrain.ProjectionPopulations {
			if p.Scope != nil && *p.Scope == spellingScope {
				index = i
				break
			}
		}
		switch corruption {
		case "duplicate":
			copyBrain.ProjectionPopulations = append(copyBrain.ProjectionPopulations, copyBrain.ProjectionPopulations[index])
		case "nonintegral":
			copyBrain.ProjectionPopulations[index].Prototype[0] = .123
		case "missing-unit":
			copyBrain.ProjectionPopulations[index].Units[0].NodeID = 999999
		}
		before, err := copyBrain.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		e, err := NewSpellingExposure(ids, "probe", .8, .8)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := copyBrain.ReadLearnedSpelling(ids); err == nil {
			t.Fatal("corrupt bank was decoded")
		}
		if err := copyBrain.ValidateSpellingExposure(e); err == nil {
			t.Fatal("corrupt bank passed preflight")
		}
		if err := copyBrain.LearnSpellingAt(e, at); err == nil {
			t.Fatal("corrupt bank was extended")
		}
		after, err := copyBrain.CanonicalJSON()
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("rejected bank operation changed the neural graph")
		}
	}
}

func TestSynapticSpellingReadSerializesWithNumericalConnectionUpdates(t *testing.T) {
	k, at := NewBrain(), time.Unix(100, 0).UTC()
	ids := spellingSource(t, k, 61, at)
	teachSpelling(t, k, ids, "B", 1, at)
	bank, _, err := k.spellingBankLocked()
	if err != nil {
		t.Fatal(err)
	}
	source, target := k.Registry.GetByID(ids[0]), k.Registry.GetByID(bank[spellingCoordinate{0, 0}])
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			k.ConnectAt(source, target, .8, .8, true, at)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			out, err := k.ReadLearnedSpelling(ids)
			if err != nil || out.Status != SpellingDecoded || out.Text != "B" {
				t.Errorf("concurrent connection update changed decoding: %+v %v", out, err)
				return
			}
		}
	}()
	wg.Wait()
}
