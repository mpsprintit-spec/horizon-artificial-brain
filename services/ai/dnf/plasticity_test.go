package dnf

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestPlasticityMutatesCanonicalDynamicSubstrate(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.20, 0.40, false)

	fabric, err := NewFabric(brain)
	if err != nil {
		t.Fatal(err)
	}

	before, err := fabric.SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 16, 6, 0, 0, 0, time.UTC)
	err = fabric.Plasticity(learning.SynapsePromotion{
		SourceNodeID: source.ID,
		TargetNodeID: target.ID,
	}, learning.Evidence{
		Weight:             0.80,
		Confidence:         0.80,
		Reliability:        0.90,
		IndependentSources: 2,
	}, now)
	if err != nil {
		t.Fatal(err)
	}

	after, err := fabric.SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if after.Weight <= before.Weight {
		t.Fatalf("expected weight to increase: before=%v after=%v", before.Weight, after.Weight)
	}
	if after.Confidence <= before.Confidence {
		t.Fatalf("expected confidence to increase: before=%v after=%v", before.Confidence, after.Confidence)
	}
	if after.Frequency <= before.Frequency {
		t.Fatalf("expected frequency to increase: before=%v after=%v", before.Frequency, after.Frequency)
	}
	if !after.LastModification.Equal(now) {
		t.Fatalf("expected modification timestamp %v, got %v", now, after.LastModification)
	}
	if fabric.Brain() != brain {
		t.Fatal("plasticity must operate on the canonical brain substrate")
	}
}

func TestPlasticityDoesNotCreateMissingDynamicSynapse(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	fabric, err := NewFabric(brain)
	if err != nil {
		t.Fatal(err)
	}

	err = fabric.Plasticity(learning.SynapsePromotion{
		SourceNodeID: source.ID,
		TargetNodeID: target.ID,
	}, learning.Evidence{Confidence: 0.90, Reliability: 0.90}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected missing dynamic synapse to be rejected")
	}
}
