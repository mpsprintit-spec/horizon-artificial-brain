package websearch

import (
	"context"
	"strings"
	"time"

	wiki "github.com/project-horizon/horizon-core/services/ai/temporary/websearch"
)

// WikipediaSearcher membatasi pencarian Horizon HANYA ke Wikipedia Indonesia,
// bukan web sembarangan.
type WikipediaSearcher struct{}

func (WikipediaSearcher) Search(ctx context.Context, query string) ([]SourceResult, error) {
	body, err := wiki.SearchWikipedia(query)
	if err != nil {
		return nil, err
	}
	result, err := wiki.ParseWikipedia(query, body)
	if err != nil {
		return nil, err
	}
	if err := wiki.ValidateWikipedia(result); err != nil {
		return nil, err
	}

	text := result.Title + " " + result.Description + " " + result.Summary
	tokens := strings.Fields(strings.ToLower(text))

	return []SourceResult{{
		Source:      result.URL,
		Reputation:  0.8,
		Tokens:      tokens,
		Confidence:  0.7,
		RetrievedAt: time.Now().UTC(),
	}}, nil
}
