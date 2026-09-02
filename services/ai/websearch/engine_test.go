package websearch

import (
	"context"
	"testing"
)

func TestShouldSearchOnlyWhenConfidenceNeedsPerception(t *testing.T) {
	e := NewEngine(nil)
	if e.ShouldSearch(0.9, 0, 0) {
		t.Fatal("high confidence should not search")
	}
	if !e.ShouldSearch(0.2, 0, 0) {
		t.Fatal("low confidence should search")
	}
	if !e.ShouldSearch(0.9, 1, 0) {
		t.Fatal("unknown token should search")
	}
}
func TestStaticSearcherMultiSource(t *testing.T) {
	e := NewEngine(StaticSearcher{Results: []SourceResult{{Source: "source one", Reputation: 0.8}, {Source: "source two", Reputation: 0.7}}})
	got, err := e.Perceive(context.Background(), "q")
	if err != nil || len(got) != 2 {
		t.Fatalf("got %d %v", len(got), err)
	}
	if len(got[0].Tokens) == 0 {
		t.Fatal("expected parsed tokens")
	}
}
