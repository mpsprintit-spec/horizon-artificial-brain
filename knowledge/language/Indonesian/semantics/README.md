# Semantics Module

## Overview

The semantics module documents Indonesian meaning: sense distinctions, lexical relations, semantic roles, ambiguity, polysemy, and semantic categories. Dictionary entries define individual words; this module defines reusable meaning principles.

## Semantic roles

- **Agent:** intentional doer, as in `Petugas membuka pintu`.
- **Patient/theme:** entity affected or moved, as in `pintu` in `membuka pintu`.
- **Experiencer:** entity that feels or perceives, as in `Saya takut`.
- **Recipient/beneficiary:** receiver or beneficiary, as in `Ibu memberi anak itu buku`.
- **Location/source/goal:** spatial roles marked by `di`, `dari`, and `ke`.

## Lexical relations

- **Synonymy:** words with overlapping meaning, such as `bisa`, `dapat`, and `mampu`, but register and nuance differ.
- **Antonymy:** contrastive pairs such as `benar`/`salah`, `aman`/`berbahaya`, `besar`/`kecil`.
- **Hyponymy:** category relation, such as `dokter` as a type of `profesi`.
- **Meronymy:** part-whole relation, such as `roda` as part of `kendaraan`.

## Polysemy and ambiguity

Some forms have multiple related or unrelated senses. `Bisa` can mean ability or venom. Context, word class, and neighboring words determine interpretation:

- `Saya bisa datang.` — ability.
- `Bisa ular berbahaya.` — venom.

Ambiguous entries should be documented in the dictionary with separate senses and referenced here only for the general interpretation strategy.

## Semantic categories

Useful categories for future entries include:

- people and social roles: `dokter`, `guru`, `keluarga`.
- safety and risk: `aman`, `bahaya`, `darurat`.
- cognition and communication: `tahu`, `pikir`, `jelas`, `tanya`.
- movement and location: `pergi`, `datang`, `di`, `ke`, `dari`.
- time and aspect: `sekarang`, `nanti`, `sudah`, `akan`.
- quantity and degree: `banyak`, `sedikit`, `sangat`, `cukup`.

## Meaning and context

Literal meaning can be modified by pragmatics. `Bisa bantu saya?` literally asks ability, but commonly functions as a polite request. See `../pragmatics/README.md` and `../conversation/README.md` for intent and interaction patterns.


## State and event representation

A lexical sense should preserve more than a short gloss when the word describes an action, change, relation, or condition. Where applicable, record:

- participants and semantic roles (agent, theme/patient, experiencer, recipient, instrument, source, goal, and location);
- the initial state and the conditions under which the expression applies;
- the event, action, or transition and its temporal order;
- the resulting state, including what is not guaranteed by the expression;
- contextual cues, alternative readings, presuppositions, and possible implicatures;
- epistemic status and review requirements for the analysis itself.

The cross-domain frame format is defined in `../../schemas/knowledge-frame-v1.schema.json`. The initial examples in `../../corpus/seed-v1.json` distinguish physical `membuka` from institutional and metaphorical uses. These entries are model-synthesized and unverified, so they are a representation pilot rather than authoritative dictionary definitions.
