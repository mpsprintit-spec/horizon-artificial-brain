package websearch

import (
	"encoding/json"
	"errors"
	"strings"
)

// ============================================================================
// Wikipedia Response
// ============================================================================

type WikipediaResponse struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	DisplayTitle string `json:"displaytitle"`
	Description string `json:"description"`
	Extract     string `json:"extract"`

	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}

// ============================================================================
// Horizon Result
// ============================================================================

type WikipediaResult struct {
	Word        string
	Title       string
	Description string
	Summary     string
	URL         string
}

// ============================================================================
// Parse Wikipedia JSON
// ============================================================================

func ParseWikipedia(word string, body []byte) (*WikipediaResult, error) {

	var response WikipediaResponse

	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Title == "" {
		return nil, errors.New("article not found")
	}

	result := &WikipediaResult{
		Word:        word,
		Title:       CleanText(response.Title),
		Description: CleanText(response.Description),
		Summary:     CleanText(response.Extract),
		URL:         response.ContentURLs.Desktop.Page,
	}

	return result, nil
}

// ============================================================================
// Clean Text
// ============================================================================

func CleanText(text string) string {

	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.TrimSpace(text)

	return text
}

// ============================================================================
// Validate Result
// ============================================================================

func ValidateWikipedia(result *WikipediaResult) error {

	if result == nil {
		return errors.New("result is nil")
	}

	if result.Title == "" {
		return errors.New("title empty")
	}

	if result.Summary == "" {
		return errors.New("summary empty")
	}

	return nil
}
