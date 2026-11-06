package knowledge

import (
	"math"
	"path/filepath"
	"sync"
	"testing"
)

func TestRegistryOrderedTraversalGrowthAndRestore(t *testing.T) {
	brain := NewBrain()
	check := func() {
		t.Helper()
		nodes := brain.Registry.Nodes()
		visited := []*ConceptNode{}
		brain.RLock()
		brain.Registry.VisitNodes(func(n *ConceptNode) { visited = append(visited, n) })
		brain.RUnlock()
		if len(nodes) != len(visited) {
			t.Fatal("stale membership")
		}
		for i, n := range nodes {
			if visited[i] != n || n != brain.Registry.GetByID(n.ID) || (i > 0 && nodes[i-1].ID >= n.ID) {
				t.Fatal("noncanonical or unordered index")
			}
		}
		if len(nodes) > 0 {
			nodes[0] = nil
			if brain.Registry.Nodes()[0] == nil {
				t.Fatal("Nodes exposed index slice")
			}
		}
	}
	check()
	for i := 1; i <= 100; i++ {
		if _, _, err := brain.Registry.GetOrCreateRepresentation(NewNeuralVector([]float64{math.Sin(float64(i) * .02), math.Cos(float64(i) * .02)}), 1); err != nil {
			t.Fatal(err)
		}
		check()
	}
	path := filepath.Join(t.TempDir(), "brain.json")
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}
	old := brain.Registry.Nodes()[0]
	if err := brain.Load(path); err != nil {
		t.Fatal(err)
	}
	check()
	if brain.Registry.Nodes()[0] == old {
		t.Fatal("restore retained old registry address")
	}
}

func TestRegistryOrderedTraversalConcurrentGrowth(t *testing.T) {
	r := NewNeuralRegistry()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 300; i++ {
			if _, _, err := r.GetOrCreateRepresentation(NewNeuralVector([]float64{math.Sin(float64(i) * .02), math.Cos(float64(i) * .02)}), 1); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 300; i++ {
		last := NodeID(0)
		r.VisitNodes(func(n *ConceptNode) {
			if n.ID <= last {
				t.Error("unstable traversal")
			}
			last = n.ID
		})
	}
	wg.Wait()
	if len(r.Nodes()) != 300 {
		t.Fatal("lost neurons")
	}
}
