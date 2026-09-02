# Indonesian Dictionary Architecture

## Overview

This document defines the structural foundation for the Indonesian Dictionary module. It describes how lexical entries should be organized, named, linked, versioned, and reviewed as the foundation grows.

The dictionary must support Horizon's knowledge-centered purpose: reliable lexical understanding that helps people interpret language in context. It should remain human-reviewable, cautious about ambiguity, and connected to the surrounding Indonesian language modules instead of duplicating their content.

## Scope

This architecture covers standards for future dictionary storage under `knowledge/language/Indonesian/dictionary/`, including entry organization, alphabet grouping, category grouping, metadata, cross-references, semantic links, pronunciation notes, examples, versioning, and scalability.

This architecture does not create a complete lexical dataset, define grammar rules, define semantic theory, store example corpora, or change the wider repository architecture. Representative vocabulary belongs in alphabet files such as `a.md` and `b.md`.

## Design principles

Future dictionary work must follow these principles:

- **One lexical identity per entry:** each entry file should represent one stable lemma or lexical form identity.
- **Meaning before labels:** parts of speech, domains, and registers support interpretation, but definitions and usage context explain why an entry matters.
- **Context before action:** ambiguous forms must point reviewers to the relevant sense, register, domain, and examples before use.
- **Cross-reference instead of duplication:** grammar, semantics, examples, idioms, abbreviations, and pragmatics remain authoritative for their own concepts.
- **Human-reviewable increments:** entries should be small, explicit, and easy to review in pull requests.
- **Stable identifiers:** entry and sense identifiers should remain stable after publication so future references do not break.
- **Dataset restraint:** this module should grow through reviewed lexical entries, not generated bulk word lists.

## Proposed directory architecture

The current foundation uses compact alphabet files for representative entries. Future contributors may add the directories below when entry volume requires one file per lexical identity:

```text
knowledge/language/Indonesian/dictionary/
├── README.md
├── architecture.md
├── entries/
│   ├── a/
│   ├── b/
│   ├── c/
│   └── ...
├── categories/
│   ├── parts-of-speech.md
│   ├── registers.md
│   ├── domains.md
│   └── relationship-types.md
├── indexes/
│   ├── alphabet.md
│   ├── roots.md
│   ├── parts-of-speech.md
│   └── semantic-links.md
└── templates/
    └── entry-template.md
```

These directories are a future storage standard for larger reviewed vocabulary sets. They should be created only when entry volume, category standards, indexes, or templates require them.

## Entry organization

At larger scale, each dictionary entry should be stored as one Markdown file. The file should document a single lexical identity and use structured headings so humans can review it and tools can parse it later.

### Entry file naming

Entry file names should be lowercase, hyphen-separated, and based on the normalized entry key. Use predictable suffixes only when needed to distinguish homographs.

Recommended pattern:

```text
entries/<first-letter>/<entry-key>.md
entries/<first-letter>/<entry-key>--<qualifier>.md
```

The `<first-letter>` directory uses the normalized first Latin letter of the entry key. The optional `<qualifier>` may identify a part of speech, etymology, domain, or homograph number when one spelling has separate lexical identities.

### Entry identity

Every entry should contain:

- **Entry ID:** stable machine-readable identifier, such as `id-dict-<entry-key>` or `id-dict-<entry-key>-<qualifier>`.
- **Word:** the displayed lexical form.
- **Normalized key:** lowercase lookup key used for sorting and cross-references.
- **Root word:** base or root form when applicable.
- **Variant forms:** spelling or orthographic variants when they belong to the same identity.
- **Homograph handling:** links to separate entry IDs when the same spelling has unrelated lexical identities.

Root words should be referenced, not redefined. Morphological behavior belongs primarily in the grammar module and should be summarized only when needed for dictionary interpretation.

## Alphabet organization

Alphabet grouping is an access strategy, not a meaning strategy. Future entries should be grouped by normalized initial letter for reviewable file sizes and stable navigation.

Rules:

- Use one directory per initial letter under `entries/`.
- Sort entries by normalized key within each alphabet index.
- Preserve the displayed word in the entry even when the normalized key differs for sorting.
- Treat nonstandard symbols, numerals, or punctuation-led forms as special cases documented in an index before adding them.
- Do not create empty alphabet directories in advance.

An alphabet index may later summarize entry IDs and file paths, but the entry file remains the authoritative record.

## Category organization

Categories describe how entries can be filtered, reviewed, and connected. They should be documented as controlled vocabularies before broad use.

Core category groups:

- **Part of speech:** lexical class such as noun, verb, adjective, adverb, particle, pronoun, numeral, preposition, conjunction, interjection, affix, or multiword expression when applicable.
- **Register:** formal, neutral, informal, colloquial, literary, archaic, technical, or other documented usage level.
- **Domain:** field of usage, such as everyday language, law, medicine, technology, education, religion, or regional culture.
- **Status:** active, rare, historical, deprecated, disputed, or needs review.
- **Relationship type:** synonymy, antonymy, derivation, variant, related term, hypernym, hyponym, meronym, holonym, idiomatic relation, and semantic reference.

Categories should not duplicate definitions. They should provide consistent labels, short descriptions, and review rules.

## Entry field standard

Each dictionary entry should use the following fields. Required fields establish identity and minimum interpretation; optional fields should be included only when evidence supports them.

### Word

The canonical displayed form of the entry. It should preserve standard orthography and casing where relevant.

### Root word

The root, base, or lemma from which the entry is derived, if applicable. Use an entry ID cross-reference when the root has its own entry. If the root relationship is uncertain, mark it as `needs review` instead of asserting it.

### Part of speech

One or more controlled part-of-speech labels. If a word belongs to multiple parts of speech, document each as a separate sense group or clearly linked sense, not as an unexplained list.

### Pronunciation

Pronunciation should be documented only with a consistent notation standard. Future entries may include:

- phonemic transcription when available and reviewed;
- syllable division when useful;
- stress or emphasis notes when relevant;
- audio reference IDs if the examples or media modules later define them;
- regional pronunciation notes when they are important for interpretation.

Pronunciation notes should avoid unsupported precision. If pronunciation varies by region or register, document the variation and evidence status.

### Meaning

A concise summary of the lexical meaning. This is a human-readable overview and should not replace structured sense definitions.

### Definitions

Definitions should be grouped into numbered senses. Each sense should contain:

- **Sense ID:** stable identifier, such as `<entry-id>#sense-1`.
- **Definition:** concise explanation in English for repository consistency.
- **Indonesian gloss:** optional short Indonesian explanation when needed for linguistic precision.
- **Part of speech:** sense-specific label when different senses have different lexical behavior.
- **Register and domain:** only when relevant.
- **Usage constraints:** notes about context, collocation, politeness, formality, or restrictions.
- **Cross-references:** links to related entries or other modules.

Multiple meanings should be represented as separate senses, not as a single paragraph. Closely related subsenses may be nested only when the parent sense remains clear.

### Usage

Usage explains how the entry behaves in context. It may include register, domain, collocation, common construction patterns, cautions, or pragmatic constraints. Detailed grammatical rules should be linked to the grammar or syntax module rather than duplicated.

### Synonyms

Synonyms should be listed by entry ID and, when possible, by sense ID. A synonym relation should identify whether it is exact, near, contextual, register-specific, or domain-specific.

### Antonyms

Antonyms should follow the same standard as synonyms. They should be sense-specific because a word with multiple meanings may have different opposites in different contexts.

### Related terms

Related terms include derivations, compounds, variants, abbreviations, idioms, and closely associated words. Each relationship should include a type and target reference.

### Examples

Dictionary entries should not become an examples database. An entry may include short illustrative examples only when needed to clarify a sense, and it should prefer references to the examples module when a reusable example record exists.

Each example reference should identify:

- target example ID or file path;
- linked sense ID;
- what the example demonstrates;
- source status, such as constructed, cited, or reviewed.

### Semantic references

Semantic references connect senses to broader meaning structures. Use them for relations that belong in semantics, such as semantic role, entailment, metaphor, taxonomy, frame, or ambiguity notes. The semantic module remains authoritative for semantic concepts.

### Notes

Notes should capture reviewer-relevant information that does not fit structured fields. Notes must not contain hidden definitions, unreviewed bulk lists, or content that belongs in another module.

### Metadata

Metadata supports review, provenance, versioning, and future tooling. See the metadata structure below.

## Metadata structure

Each future entry should include a metadata block near the top or bottom of the file. The block may be represented as Markdown fields now and converted to front matter later if tooling requires it.

Recommended metadata fields:

```text
Entry ID: <stable-entry-id>
Status: draft | reviewed | deprecated | needs-review
Language: Indonesian
Module: dictionary
Created: YYYY-MM-DD
Updated: YYYY-MM-DD
Version: major.minor.patch
Maintainers: <team-or-role>
Evidence level: documented | reviewed | provisional | needs-source
Source status: original-review | cited-source | constructed-example | imported-with-license
Related modules: grammar, semantics, examples, idioms, abbreviations, pragmatics, syntax
```

Metadata rules:

- Use ISO dates.
- Update `Updated` whenever entry content changes.
- Increment `Version` when definitions, senses, references, or metadata change.
- Record source and license information before importing externally authored lexical data.
- Mark uncertain content as provisional or needs review instead of presenting it as final.

## Cross-reference strategy

Cross-references keep the dictionary connected without copying content across modules.

### Reference format

Use stable references with readable labels:

```text
- Dictionary entry: `id-dict-<entry-key>`
- Dictionary sense: `id-dict-<entry-key>#sense-<number>`
- Module document: `knowledge/language/Indonesian/<module>/<file>.md`
- Section anchor: `knowledge/language/Indonesian/<module>/<file>.md#section-name`
```

### Reference rules

- Reference the most specific stable target available.
- Prefer sense-level links for synonyms, antonyms, examples, and semantic relations.
- Link to grammar for morphology and part-of-speech behavior.
- Link to syntax for phrase, clause, or sentence structure.
- Link to semantics for meaning relations and interpretation rules.
- Link to pragmatics for politeness, speech acts, and social context.
- Link to examples for reusable examples.
- Link to idioms or expressions for multiword meanings that are not ordinary dictionary senses.
- Link to abbreviations for shortened forms when the abbreviation identity is primary there.

Do not duplicate another module's explanation. A dictionary entry may summarize why the reference matters and then point to the authoritative module.

## Related-word and semantic-link strategy

Relationships must be typed and directional when direction matters.

Recommended relationship record:

```text
Relation type: synonym | antonym | root | derived-form | variant | related-term | hypernym | hyponym | meronym | holonym | idiom-component | semantic-reference
Source: <entry-id-or-sense-id>
Target: <entry-id-or-sense-id-or-module-reference>
Scope: entry | sense | usage | pronunciation | spelling
Strength: exact | near | contextual | weak | needs-review
Note: <short explanation>
```

Rules:

- Use sense-level relationships for meaning-sensitive links.
- Use entry-level relationships only for spelling variants, roots, and broad lexical associations.
- Avoid circular explanations where two entries define each other without an independent definition.
- Mark speculative or disputed relationships as `needs-review`.
- Keep semantic taxonomy and interpretation rules in the semantics module; dictionary entries should reference them.

## Versioning strategy

Dictionary architecture and future entries should use explicit versions.

### Architecture versions

This document is the architecture authority for the dictionary module. Structural changes should update an architecture version note in the pull request summary and explain migration expectations.

### Entry versions

Future entries should use semantic versioning:

- **Major:** sense identity changes, entry split or merge, removed meaning, or breaking reference changes.
- **Minor:** new sense, new relationship, new pronunciation section, or substantial usage expansion.
- **Patch:** typo correction, formatting, metadata update, or non-semantic clarification.

Deprecated entries should remain as redirect records when other entries may reference them. Do not delete reviewed entry files without a documented migration plan.

## Future scalability

The dictionary should scale by structure before volume.

Scalability rules:

- Add entries incrementally by reviewed pull request.
- Keep alphabet directories small enough for easy review; split with additional index files only when necessary.
- Use indexes as navigation aids, not as duplicate data stores.
- Keep stable entry IDs independent of file paths when possible so files can move during future reorganization.
- Require controlled categories before applying a new category broadly.
- Prefer references to examples, semantics, and grammar over copying large explanations into entries.
- Record source and license metadata before importing external lexical data.
- Avoid generated vocabulary dumps unless a separate governance document approves import quality, licensing, deduplication, and review workflow.

## Contributor checklist

Before adding or modifying future dictionary content, contributors should verify:

- the entry has a stable ID, normalized key, status, and version;
- each meaning is represented by a separate sense ID;
- part-of-speech labels use controlled categories;
- pronunciation information follows a consistent notation and evidence standard;
- examples are short or referenced from the examples module;
- synonyms, antonyms, related terms, and semantic links are typed and sense-specific where needed;
- cross-references point to authoritative modules instead of duplicating them;
- uncertain content is marked as provisional or needs review;
- no file has been renamed or removed without an explicit migration plan;
- the change remains consistent with the Indonesian Language README and The Book of Horizon.
