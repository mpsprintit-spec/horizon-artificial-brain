package runtime

import (
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"testing"
)

func TestMetricLearnsReferenceFallbackWithoutForgettingNamedBinding(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	phase := 0
	for h, field := range []int{49, 50, 48} {
		rows := []dialogueTrainingRow{}
		for rep := 0; rep < 2; rep++ {
			for _, v := range []float64{-.75, -.5, -.25, 0, .25, .5, .75, 1} {
				rows = append(rows, dialogueTrainingRow{Fields: map[int]float64{field: v}, Value: v})
			}
		}
		dialogueFitStage(t, r, 17+h, rows, rows, dialoguePrecision, &phase, 3)
	}
	base := []dialogueTrainingRow{}
	combined := []dialogueTrainingRow{}
	for _, prior := range []float64{0, .25, .5, .75} {
		for _, ref := range []float64{.25, .5, .75} {
			row := dialogueTrainingRow{Fields: map[int]float64{48: ref / 2, 49: .5, 50: prior}, Value: ref}
			base = append(base, row)
			combined = append(combined, row, dialogueTrainingRow{Fields: map[int]float64{48: 0, 49: 0, 50: prior}, Value: prior})
		}
	}
	dialogueFitStage(t, r, 7, base, base, dialoguePrecision, &phase, 5)
	dialogueFitStage(t, r, 7, combined, combined, dialoguePrecision, &phase, 8)
	for _, set := range [][]dialogueTrainingRow{base, combined} {
		n, e := dialogueRowsPass(t, r, 7, set, dialoguePrecision)
		if n != len(set) {
			t.Fatal(fmt.Sprintf("conditional binding failed %d/%d, max error %g", n, len(set), e))
		}
	}
}
