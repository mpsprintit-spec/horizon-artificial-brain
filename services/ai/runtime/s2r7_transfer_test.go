//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestS2R7DevelopmentTransfer(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	phase, seed, _ := s2r7PhaseSeed(t)
	if phase != "development" {
		t.Fatal("development transfer only")
	}
	cp, wal := s2r3Source(root)
	base := filepath.Join(work, phase, fmt.Sprintf("seed-%d", seed))
	readers := map[string]*BrainRuntime{}
	descs := map[string]string{}
	for _, arm := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		readers[arm] = orderedFixtureClone(t, filepath.Join(base, arm, "final.json"))
		descs[arm] = "D3"
		if arm == "A" {
			descs[arm] = "D0"
		}
		if arm == "B" {
			descs[arm] = "D1"
		}
	}
	br := orderedFixtureClone(t, filepath.Join(base, "bootstrap/final.json"))
	for _, d := range []string{"D0", "D1", "D2", "D3", "D4"} {
		readers[d] = br
		descs[d] = d
	}
	for ci, condition := range []string{"unseen", "shifted"} {
		dir := filepath.Join(base, "transfer-"+condition)
		observer, w := s2r2Fork(t, cp, wal, filepath.Join(dir, "native-observer"))
		s2r7Acquire(t, observer, readers, pop, filepath.Join(dir, "acquisition"), 274000000+int64(seed*1000+ci*1000000), 8, 16, ci == 1, true, descs, false)
		s1Check(t, w.Close())
	}
}
