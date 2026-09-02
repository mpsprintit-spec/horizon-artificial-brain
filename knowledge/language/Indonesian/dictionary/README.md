# Dictionary Module

## Overview

The dictionary module documents Indonesian lexical entries. It provides a scalable structure for lemma identity, pronunciation, part of speech, meanings, usage, related words, and examples.

## Structure

- `architecture.md` defines the long-term dictionary design.
- `a.md`, `b.md`, and later alphabet files store representative entries by initial letter.
- New alphabet files may be added when reviewed entries require them.

## Entry format

Each entry should include:

- **Lemma:** Indonesian headword.
- **Pronunciation:** broad practical pronunciation using Indonesian spelling guidance or IPA when useful.
- **Part of speech:** noun, verb, adjective, adverb, pronoun, preposition, conjunction, particle, numeral, or interjection.
- **Register:** formal, neutral, informal, colloquial, regional, or domain-specific.
- **Meaning:** concise English meaning with sense numbers when needed.
- **Usage:** contextual guidance and constraints.
- **Related words:** derivations, synonyms, antonyms, or semantically related terms.
- **Examples:** short Indonesian examples with English glosses.
- **Cross-references:** grammar, semantics, pragmatics, expressions, idioms, abbreviations, or examples when relevant.

## Initial vocabulary coverage

The current foundation begins with high-value, reusable entries: function words, common verbs, social terms, safety-relevant vocabulary, and language-learning anchors. It intentionally does not attempt to be complete.

## Cross-references

Use `../grammar/README.md` for word class and affix behavior, `../semantics/README.md` for meaning relations, and `../examples/README.md` for sentence-level demonstrations.
