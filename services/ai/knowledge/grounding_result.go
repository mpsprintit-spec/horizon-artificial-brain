package knowledge

// GroundingResult describes how a numeric observation maps onto the shared
// neural substrate. It intentionally carries structural status only; it does
// not assert semantic truth or authorization.
type GroundingResult struct {
	NodeID  string
	Existing bool
	Similarity float64
}
