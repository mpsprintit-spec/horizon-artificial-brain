package websearch

import (
	"context"
	"strings"
	"time"
)

type SourceResult struct {
	Source      string
	Reputation  float64
	Tokens      []string
	Confidence  float64
	RetrievedAt time.Time
}
type Searcher interface {
	Search(ctx context.Context, query string) ([]SourceResult, error)
}
type Engine struct {
	Searcher      Searcher
	MinConfidence float64
}

func NewEngine(searcher Searcher) *Engine { return &Engine{Searcher: searcher, MinConfidence: 0.55} }
func (e *Engine) ShouldSearch(confidence float64, unknownCount int, conflicts int) bool {
	return confidence < e.MinConfidence || unknownCount > 0 || conflicts > 0
}
func (e *Engine) Perceive(ctx context.Context, query string) ([]SourceResult, error) {
	if e.Searcher == nil {
		return nil, nil
	}
	return e.Searcher.Search(ctx, query)
}

type StaticSearcher struct{ Results []SourceResult }

func (s StaticSearcher) Search(ctx context.Context, query string) ([]SourceResult, error) {
	out := make([]SourceResult, 0, len(s.Results))
	for _, r := range s.Results {
		if len(r.Tokens) == 0 {
			r.Tokens = strings.Fields(strings.ToLower(r.Source))
		}
		if r.RetrievedAt.IsZero() {
			r.RetrievedAt = time.Now().UTC()
		}
		out = append(out, r)
	}
	return out, nil
}
