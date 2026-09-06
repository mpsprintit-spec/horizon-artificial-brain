package neural

import "testing"

func TestDeterministicReplay(t *testing.T) {
	cfg := DefaultConfig()
	a, err := New(cfg); if err != nil { t.Fatal(err) }
	b, err := New(cfg); if err != nil { t.Fatal(err) }
	idsA := make([]NeuronID, 3); idsB := make([]NeuronID, 3)
	for i := range idsA { idsA[i], _ = a.AddNeuron(); idsB[i], _ = b.AddNeuron() }
	for i := 0; i < 2; i++ { if _, err := a.Connect(idsA[i], idsA[i+1], 0.4); err != nil { t.Fatal(err) }; if _, err := b.Connect(idsB[i], idsB[i+1], 0.4); err != nil { t.Fatal(err) } }
	for step := 0; step < 50; step++ {
		input := float64((step % 3) + 1)
		if err := a.Inject(idsA[0], input); err != nil { t.Fatal(err) }
		if err := b.Inject(idsB[0], input); err != nil { t.Fatal(err) }
		a.Tick(); b.Tick(); a.Learn(0.5); b.Learn(0.5)
	}
	for i := range idsA { na, _ := a.Neuron(idsA[i]); nb, _ := b.Neuron(idsB[i]); if na != nb { t.Fatalf("neuron %d diverged: %#v != %#v", i, na, nb) } }
	for id := SynapseID(1); id <= 2; id++ { sa, oka := a.Synapse(id); sb, okb := b.Synapse(id); if !oka || !okb || sa != sb { t.Fatalf("synapse %d diverged", id) } }
}
