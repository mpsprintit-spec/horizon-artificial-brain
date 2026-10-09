# Horizon Knowledge Corpus

## Purpose

This directory is the canonical human-reviewable source for structured knowledge intended to be processed by Horizon. It is not a transcript archive and must not be populated from personal conversation history. Knowledge should be drawn from general linguistic, mathematical, scientific, engineering, and empirical material, with explicit provenance and uncertainty.

The corpus stores more than labels or isolated facts. When relevant, each entry preserves entities, roles, initial conditions, events or transformations, resulting states, context, constraints, relations, uncertainty, and evidence status.

## Domains

- `language/Indonesian/`: Indonesian lexical, morphological, syntactic, semantic, pragmatic, and discourse knowledge.
- `mathematics/`: arithmetic, algebra, formal logic, sets, geometry, linear algebra, calculus, probability, statistics, discrete mathematics, and dynamical systems.
- `physical-sciences/`: physics and other physical-science models, with equations, coordinate conventions, initial conditions, assumptions, and limits of applicability.
- `biology-neuroscience/`: biological structures, functions, interactions, mechanisms, stimulus-response relationships, and research findings.
- `computing-engineering/`: algorithms, logic, architecture, information flow, input-output behavior, constraints, and design principles.
- `empirical-reasoning/`: observations, evidence, hypotheses, predictions, outcomes, prediction errors, confidence, and revision conditions.
- Language interaction patterns such as questions, answers, commands, corrections, ambiguity, and shifts in understanding belong in the Indonesian language modules and may cross-reference other domains.

## Representation rules

1. Preserve a concept's relevant state and relations; do not reduce an entry to a word-to-number pair.
2. Separate the source statement from the interpretation or inference derived from it.
3. Keep ambiguity explicit. Do not silently choose one sense when context does not disambiguate it.
4. Distinguish established findings, source claims, model-generated synthesis, and unverified hypotheses.
5. Record boundary conditions and counterexamples where they affect validity.
6. Cross-reference a concept's canonical definition rather than duplicating it across modules.
7. A file's presence in this corpus does not mean Horizon has imported, learned, or validated it. Runtime ingestion and learning require separate tests.

## Initial implementation

- `schemas/knowledge-frame-v1.schema.json` defines a source-independent structured frame for concepts, events, conditions, relations, and evidence.
- `schemas/knowledge-corpus-v1.schema.json` defines the corpus envelope and references the individual-frame schema.
- `corpus/seed-v1.json` is a small cross-domain pilot covering Indonesian lexical meaning, mathematics, physics, biology, computing, and empirical reasoning. Its model-synthesized entries are explicitly marked unverified and are intended to test representation shape, not serve as a fully reviewed reference work.

## Relationship to runtime

The Go package `services/ai/knowledge` now has a minimal adapter:
- `DecodeKnowledgeFrameCorpus` reads the corpus envelope, rejects duplicate IDs and malformed core metadata, and retains each complete record as raw JSON.
- `KnowledgeDocuments` maps a record's initial state, mechanism/transition, and resulting state into the existing before/change/after input model.
- `KnowledgeDatum.SemanticFrame` and `SymbolExposure.SemanticFrame` preserve the complete source frame in canonical developmental experience data.
- `BrainRuntime.ProcessKnowledgeFrameCorpus` passes records through the existing runtime processing path.

Important boundary: the numeric representations are currently generated from the mapped state-summary strings. Preserving a full frame in canonical metadata does not mean the neural substrate has independently interpreted every role, relation, condition, or limitation in that frame. That requires additional structural grounding and behavior-level tests. The adapter's minimum structural checks are not a substitute for full JSON Schema validation or source verification.

## Growth and review

Expand in small, reviewable batches. For each batch:
- validate JSON against the schema;
- review factual claims and source provenance;
- test that meaning-bearing fields survive compilation and persistence;
- test relevant runtime behavior with positive and negative controls;
- report separately what was authored, imported, and behaviorally demonstrated.
