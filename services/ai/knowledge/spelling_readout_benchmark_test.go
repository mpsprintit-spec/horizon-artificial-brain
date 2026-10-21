package knowledge

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkLearnedSpellingReadout(b *testing.B) {
	for _, words := range []int{8, 32} {
		b.Run(fmt.Sprintf("words-%d", words), func(b *testing.B) {
			k := NewBrain()
			at := time.Unix(100, 0).UTC()
			rng := rand.New(rand.NewSource(20261021))
			var populations [][]NodeID
			var spellings []string
			for i := 0; i < words; i++ {
				values := make([]float64, 16)
				for j := range values {
					values[j] = rng.Float64()*2 - 1
				}
				p, err := k.ProjectVectorPopulationAt(NewNeuralVector(values), .999999, 4, at)
				if err != nil {
					b.Fatal(err)
				}
				var ids []NodeID
				for _, unit := range p.Units {
					ids = append(ids, unit.NodeID)
				}
				populations = append(populations, ids)
				text := fmt.Sprintf("w%05d", i)
				spellings = append(spellings, text)
				e, err := NewSpellingExposure(ids, text, .8, .8)
				if err != nil {
					b.Fatal(err)
				}
				for j := 0; j < 4; j++ {
					if err := k.LearnSpellingAt(e, at); err != nil {
						b.Fatal(err)
					}
				}
			}
			canonical, err := k.CanonicalJSON()
			if err != nil {
				b.Fatal(err)
			}
			synapses := 0
			for _, node := range k.Registry.Nodes() {
				for _, list := range node.Synapses {
					synapses += len(list)
				}
			}
			units := len(k.Registry.Nodes())
			bank, _, err := k.spellingBankLocked()
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, err := k.ReadLearnedSpelling(populations[i%words])
				if err != nil || out.Status != SpellingDecoded || out.Text != spellings[i%words] {
					b.Fatal("learned readout failed", out, err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(units), "units")
			b.ReportMetric(float64(synapses), "synapses")
			b.ReportMetric(float64(len(bank)), "output-cells")
			b.ReportMetric(float64(len(canonical)), "canonical-bytes")
			b.ReportMetric(float64(len(k.Registry.Nodes())-units), "new-units")
		})
	}
}
