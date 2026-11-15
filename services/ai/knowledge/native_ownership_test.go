package knowledge

import "testing"

func TestNativeOwnershipHandlesNoncontiguousAddresses(t *testing.T) {
	b := NewBrain()
	b.Lock()
	defer b.Unlock()
	empty := b.NativeOwnershipLocked()
	if empty.Enabled() || empty.Has(0) || empty.Has(100) {
		t.Fatal("unconfigured ownership")
	}
	b.BrainState.NativeControl = &NativeControlState{Bias: 100, Features: []NodeID{2, 19}, Outputs: [][]NodeID{{7, 40}, {505, 8}}}
	o := b.NativeOwnershipLocked()
	expected := map[NodeID]bool{100: true, 2: true, 19: true, 7: true, 40: true, 505: true, 8: true}
	for id := NodeID(0); id < 600; id++ {
		if o.Has(id) != expected[id] {
			t.Fatalf("incorrect role ownership for address %d", id)
		}
	}
}
