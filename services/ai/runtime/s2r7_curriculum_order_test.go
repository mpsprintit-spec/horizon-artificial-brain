//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestS2R7CurriculumOrder(t *testing.T) {
	_, work, _ := s2r7Paths(t)
	for _, seed := range []int{37, 211} {
		base := filepath.Join(work, "development", fmt.Sprintf("seed-%d", seed))
		dir := filepath.Join(base, "curriculum-order-check")
		r, w := s2r2Fork(t, filepath.Join(base, "bootstrap/final.json"), filepath.Join(base, "bootstrap/events.jsonl"), dir)
		var rows, aux, dev []s2r7Row
		s1Read(t, filepath.Join(base, "utility-acquisition/D3-rows.json"), &rows)
		s1Read(t, filepath.Join(base, "auxiliary-acquisition/D3-rows.json"), &aux)
		s1Read(t, filepath.Join(base, "development-acquisition/D3-rows.json"), &dev)
		rng := int64(271300000 + seed*1000)
		af, ac := s2r7Partition(aux, false, rng+13)
		fit, check := s2r7Partition(rows, true, rng+111)
		receipts := []*CompositionLearningReceipt{}
		rec, _ := s2r7Lesson(t, r, fmt.Sprintf("r7-order-%d-contribution", seed), af[:64], ac[:64], true, false, "contribution", rng)
		receipts = append(receipts, rec)
		for b := 0; b < 4; b++ {
			rec, _ = s2r7Lesson(t, r, fmt.Sprintf("r7-order-%d-b%d", seed, b), fit[64*b:64*(b+1)], check[64*b:64*(b+1)], true, false, "utility", rng+int64(b*100))
			receipts = append(receipts, rec)
		}
		before := s2r7ReadRows(t, r, dev, "full")
		rec, _ = s2r7Lesson(t, r, fmt.Sprintf("r7-order-%d-retrospective", seed), af[64:128], ac[64:128], true, false, "retrospective", rng+1)
		receipts = append(receipts, rec)
		after := s2r7ReadRows(t, r, dev, "full")
		var old map[string]struct {
			Prediction []float64 `json:"prediction"`
		}
		s1Read(t, filepath.Join(base, "E/curves.json"), &old)
		if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(before, old["512"].Prediction) {
			t.Fatal("auxiliary order changed prospective utility functions")
		}
		s2r7Write(t, filepath.Join(dir, "ORDER_PROOF.json"), map[string]any{"seed": seed, "prospective_before_after_retrospective_exact": true, "development_v1_v2_prospective_exact": true, "N": len(before), "predictions": after, "receipts": receipts})
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
}
