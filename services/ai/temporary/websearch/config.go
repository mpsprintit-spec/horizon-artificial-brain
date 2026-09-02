package websearch

import "time"

// ============================================================================
// Horizon Temporary WebSearch Configuration
// Temporary module.
// Can be removed safely without affecting Horizon Core.
// ============================================================================

// Wikipedia API
const WikipediaAPI = "https://id.wikipedia.org/api/rest_v1/page/summary/"

// HTTP Configuration
const (
	HTTPTimeout = 30 * time.Second
	UserAgent   = "Horizon-Core/1.0"
)

// Learning Configuration
const (
	MaxWordDepth = 100
	MaxSummary   = 4000
)

// User Response
const (
	AnswerYes = "Y"
	AnswerNo  = "N"
)

// Status
const (
	StatusFound     = "FOUND"
	StatusNotFound  = "NOT_FOUND"
	StatusApproved  = "APPROVED"
	StatusRejected  = "REJECTED"
	StatusSaved     = "SAVED"
	StatusSkipped   = "SKIPPED"
)
