# Indonesian Language Knowledge Base

## Overview

The Indonesian Language Knowledge Base provides Horizon with reviewed linguistic knowledge for understanding Indonesian in human contexts. It is organized as a modular foundation rather than a bulk corpus: each module owns one kind of knowledge and cross-references related modules when interpretation depends on more than one layer.

This foundation follows The Book of Horizon by treating language knowledge as support for safer, clearer, and more context-aware assistance. Indonesian examples are included only where they represent the language itself; explanations remain in professional English.

## Module map

- `dictionary/` owns lexical entries, lemma identity, pronunciations, meanings, usage notes, related words, and entry metadata.
- `grammar/` owns word classes, affixes, word formation, voice, pronouns, aspect markers, and grammatical rules.
- `syntax/` owns phrase, clause, sentence, word-order, and dependency patterns.
- `semantics/` owns meaning relations, roles, ambiguity, polysemy, synonymy, antonymy, and semantic categories.
- `pragmatics/` owns contextual usage, register, politeness, social distance, intent, and cultural interpretation.
- `conversation/` owns interaction patterns such as greetings, questions, requests, confirmations, corrections, agreement, disagreement, and closings.
- `expressions/` owns conventional set expressions whose meanings are mostly compositional but socially conventional.
- `idioms/` owns idiomatic expressions whose meanings are not fully predictable from their parts.
- `abreviations/` owns abbreviations, acronyms, initialisms, shortened forms, and ambiguity notes.
- `examples/` owns representative examples that demonstrate and link the other modules without becoming a corpus.

## Cross-reference rule

Each concept should be defined once in its primary module. Other modules should reference that definition rather than restating it. For example, the grammar module defines the active prefix `meN-`, the syntax module explains where the resulting verb appears in a clause, and the examples module provides short sentences that demonstrate both.

## Terminology standard

Use the following terms consistently:

- **entry:** a documented lexical item, abbreviation, expression, idiom, rule, pattern, or example.
- **meaning:** the stable interpretation of an entry in a defined context.
- **usage:** guidance about when and how an entry is appropriate.
- **register:** formality level such as formal, neutral, informal, or colloquial.
- **context:** the social, situational, or linguistic environment that changes interpretation.
- **cross-reference:** a link to another module that owns related knowledge.

## Current foundation status

This foundation contains representative knowledge, not a complete Indonesian dictionary or grammar. It is designed to be scalable: future contributors can add entries and focused files without renaming existing folders, duplicating module ownership, or changing repository architecture.
