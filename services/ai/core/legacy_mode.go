package core

// LegacyCognitionEnabled controls the temporary compatibility path that still
// uses ThinkingEngine/FSU/RelationKind. It is deliberately opt-in so legacy
// cognition cannot silently become Horizon's source of intelligence.
var LegacyCognitionEnabled bool
