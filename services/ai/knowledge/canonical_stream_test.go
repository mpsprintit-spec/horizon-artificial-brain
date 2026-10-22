package knowledge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamingCanonicalMatchesFrozenEncoder(t *testing.T) {
	for _, bootstrap := range []bool{false, true} {
		b := NewBrain()
		if bootstrap {
			b = NewBootstrapBrain()
		}
		b.BrainState.MemoryState["escaped<>&\u2028\n"] = -math.SmallestNonzeroFloat64
		// Parallel connections with identical historical sort keys but distinct
		// dynamic values exercise global-sort tie compatibility.
		for i := 0; i < 40; i++ {
			n, _, e := b.Registry.GetOrCreateRepresentation(NewNeuralVector([]float64{float64(i + 1), .1}), 1)
			if e != nil {
				t.Fatal(e)
			}
			n.Synapses = map[NodeID]SynapseList{}
			for j := 0; j < 7; j++ {
				n.Synapses[1] = append(n.Synapses[1], &Synapse{TargetID: 1, Weight: float64(j) / 10, Inhibitory: j%3 == 0})
			}
		}
		want, err := marshalCanonicalReference(b)
		if err != nil {
			t.Fatal(err)
		}
		var streamed bytes.Buffer
		err = b.WriteCanonicalJSON(&streamed)
		got := streamed.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("streaming changed canonical bytes")
		}
		var compact, gotCompact bytes.Buffer
		_ = json.Compact(&compact, want)
		if err := b.WriteCanonical(&gotCompact, false, ""); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(compact.Bytes(), gotCompact.Bytes()) {
			t.Fatal("compact canonical encoding differs")
		}
		var prefixed, gotPrefixed bytes.Buffer
		_ = json.Indent(&prefixed, want, "  ", "  ")
		if err := b.WriteCanonical(&gotPrefixed, true, "  "); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(prefixed.Bytes(), gotPrefixed.Bytes()) {
			t.Fatal("embedded canonical indentation differs")
		}
		h := sha256.Sum256(want)
		hash, e := b.CanonicalHash()
		if e != nil || hash != "sha256:"+hex.EncodeToString(h[:]) {
			t.Fatal("stream hash changed")
		}
	}
}

type rejectedCanonicalWriter struct{}

func (rejectedCanonicalWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestCanonicalStreamingPropagatesErrorsAndSaveIsAtomic(t *testing.T) {
	b := NewBootstrapBrain()
	if err := b.WriteCanonicalJSON(rejectedCanonicalWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "brain.json")
	if err := os.WriteFile(path, []byte("old-state"), 0600); err != nil {
		t.Fatal(err)
	}
	b.Registry.Nodes()[0].Representation = []float64{math.NaN()}
	if err := b.Save(path); err == nil {
		t.Fatal("invalid neural value accepted")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "old-state" {
		t.Fatal("failed encoding replaced prior state")
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".brain-memory-*.tmp"))
	if len(files) != 0 {
		t.Fatal("temporary file leaked")
	}
}
