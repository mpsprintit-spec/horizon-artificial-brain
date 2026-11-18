//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestS2R7IIDTrain(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	phase, seed, _ := s2r7PhaseSeed(t)
	if phase != "confirmatory" {
		t.Fatal("IID confirmatory only")
	}
	if _, e := os.Stat(filepath.Join(work, "TRAINING_LOCK.json")); e != nil {
		t.Fatal("IID protocol not frozen")
	}
	cp, wal := s2r3Source(root)
	base := filepath.Join(work, phase, fmt.Sprintf("seed-%d", seed))
	od := filepath.Join(base, "iid-observer")
	observer, ow := s2r2Fork(t, cp, wal, od)
	rows := s2r4Collect(t, observer, pop, od, "independent", 284000000+int64(seed*1000), 16, 16, false)
	s1Check(t, ow.Close())
	for _, mask := range []bool{false, true} {
		name := "iid"
		if mask {
			name = "iid-masked"
		}
		dir := filepath.Join(base, name)
		r, w := s2r2Fork(t, cp, wal, dir)
		s2r5ReplayObserver(t, r, filepath.Join(od, "events.jsonl"))
		receipts := []*CompositionLearningReceipt{}
		for b := 0; b < 4; b++ {
			rec, _ := s2r6HistoryLesson(t, r, s2r6Arm{Mask: mask, Coverage: true, IID: true}, fmt.Sprintf("r7-iid-%d-%s-%d", seed, name, b), s2r4Subset(rows, 4*b, 4*b+2), s2r4Subset(rows, 4*b+2, 4*b+4), pop, filepath.Join(work, "history-contract"))
			receipts = append(receipts, rec)
			s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
		}
		s2r7Write(t, filepath.Join(dir, "receipts.json"), receipts)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
}
