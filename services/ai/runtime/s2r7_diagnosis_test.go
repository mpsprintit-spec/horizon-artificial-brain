//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestS2R7R6Audit(t *testing.T) {
	archive, work := os.Getenv("HORIZON_S2R7_ARCHIVE"), os.Getenv("HORIZON_S2R7_WORK")
	if archive == "" || work == "" {
		t.Skip("isolated R6 diagnosis")
	}
	s1Check(t, os.MkdirAll(filepath.Join(work, "r6-audit"), 0700))
	for _, seed := range s2r5Seeds {
		base := filepath.Join(archive, fmt.Sprintf("r6/training/seed-%d", seed))
		r := orderedFixtureClone(t, filepath.Join(base, "C/final.json"))
		out := map[string]any{}
		for _, condition := range []string{"fit-check", "seen", "unseen", "shifted"} {
			file := filepath.Join(base, "utility-acquisition/rows.json")
			if condition != "fit-check" {
				file = filepath.Join(base, "C/eval-"+condition+"/rows.json")
			}
			var rows []s2r6UtilityRow
			s1Read(t, file, &rows)
			audits := []any{}
			for _, row := range rows {
				v, err := r.brain.S2R7Observe(row.Input)
				s1Check(t, err)
				audits = append(audits, map[string]any{"ID": row.ID, "OriginID": row.OriginID, "Stream": row.Stream, "Context": row.Context, "Utility": row.Utility, "Prediction": row.Prediction, "Representation": v, "Before": row.Before, "After": row.After, "Actual": row.Actual})
			}
			out[condition] = audits
		}
		s1Write(t, filepath.Join(work, "r6-audit", fmt.Sprintf("seed-%d.json", seed)), out)
		t.Log("audited R6", seed)
	}
}
