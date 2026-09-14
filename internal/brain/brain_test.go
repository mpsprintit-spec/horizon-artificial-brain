package brain

import (
	"testing"

	"github.com/project-horizon/horizon-core/internal/neural"
)

func TestBrainProcessesContinuousExperience(t *testing.T) {
	b,err:=New(2,2,neural.DefaultConfig());if err!=nil{t.Fatal(err)}
	before:=b.Network().StepCount()
	for i:=0;i<10;i++ {if _,err:=b.Step([]float64{2,0},1);err!=nil{t.Fatal(err)}}
	if b.Network().StepCount()!=before+10{t.Fatalf("expected persistent steps, got %d",b.Network().StepCount())}
	if b.Network().NeuronCount()!=4{t.Fatalf("unexpected neuron count")}
}
