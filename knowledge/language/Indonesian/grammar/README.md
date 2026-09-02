# Grammar Module

## Overview

The grammar module documents Indonesian word classes, morphology, affixes, word formation, voice, pronouns, and tense-aspect expression. Indonesian grammar is largely analytic: verbs do not conjugate for person, number, or tense, and grammatical meaning is often expressed through particles, affixes, word order, and context.

## Parts of speech

- **Nouns:** name people, objects, places, concepts, and events. Plurality is usually inferred from context or expressed with numerals, quantifiers, or reduplication: `buku` (book/books), `dua buku` (two books), `buku-buku` (books, various books).
- **Verbs:** express actions, processes, states, or events. Verbs do not inflect for tense. Voice and transitivity are often marked with affixes such as `meN-`, `di-`, `ber-`, and `ter-`.
- **Adjectives:** describe qualities and can function as predicates without a copula: `Rumah itu besar` (“The house is big”).
- **Adverbs:** modify verbs, adjectives, clauses, or whole utterances: `sangat` (very), `segera` (immediately), `mungkin` (maybe).
- **Pronouns:** encode person, number, social relation, and formality. Choice of `saya`, `aku`, `Anda`, `kamu`, `dia`, `beliau`, `kami`, and `kita` is pragmatic as well as grammatical.
- **Prepositions:** introduce locations, directions, sources, instruments, or relations: `di`, `ke`, `dari`, `dengan`, `untuk`.
- **Conjunctions:** connect words, phrases, or clauses: `dan`, `atau`, `tetapi`, `karena`, `jika`.
- **Particles:** add focus, emphasis, politeness, or discourse meaning: `-lah`, `-kah`, `pun`, `dong`, `kok`, `ya`.

## Sentence structure

The unmarked Indonesian clause pattern is subject-predicate, often realized as SVO for transitive verbal clauses:

- `Saya membaca buku.` — subject + verb + object.
- `Anak itu pintar.` — subject + adjective predicate.
- `Mereka di rumah.` — subject + prepositional predicate.

See `../syntax/README.md` for phrase and clause ordering.

## Tense, aspect, and modality

Indonesian does not mark tense by verb inflection. Time is expressed by context, time adverbs, or aspect/modal markers:

- `sudah` — completed or already: `Saya sudah makan`.
- `sedang` — ongoing: `Dia sedang bekerja`.
- `akan` — future or intended: `Kami akan berangkat`.
- `belum` — not yet: `Mereka belum datang`.
- `bisa`, `dapat`, `mampu` — ability or possibility.
- `harus`, `perlu`, `sebaiknya` — obligation, need, or recommendation.

## Affixes and word formation

Indonesian forms many words through affixation. The same root may produce several grammatical and semantic forms.

| Pattern | General function | Example | Meaning |
| --- | --- | --- | --- |
| `meN-` | active transitive/intransitive verb | `membaca` | to read |
| `di-` | passive verb | `dibaca` | to be read |
| `ber-` | intransitive, have/wear/do | `berjalan` | to walk |
| `ter-` | accidental, stative, superlative | `terbuka` | open/opened unintentionally |
| `peN-` | agent/instrument noun | `pembaca` | reader |
| `per-...-an` | abstract/process/place noun | `perjalanan` | journey |
| `ke-...-an` | abstract state or excessive condition | `kesehatan` | health |
| `-kan` | causative, applicative, benefactive | `menyalakan` | to turn on |
| `-i` | locative or repetitive applicative | `mengisi` | to fill |
| `-an` | result, object, or collective noun | `makanan` | food |

`meN-` changes shape according to the first sound of the root, as in `baca` → `membaca` and `tulis` → `menulis`. Document specific derivations in dictionary entries, not by generating exhaustive lists here.

## Active and passive voice

- **Active voice:** the actor is prominent: `Petugas membuka pintu` (“The officer opens the door”).
- **Passive voice with `di-`:** the patient/theme is prominent: `Pintu dibuka oleh petugas` (“The door is opened by the officer”).
- **Agentless passive:** common when the actor is unknown or irrelevant: `Pintu dibuka` (“The door was/is opened”).
- **First/second-person passive-like order:** Indonesian often places patient before a bare verb phrase with a pronoun actor: `Buku itu saya baca` (“I read/read the book,” literally “That book I read”).

## Pronouns and social meaning

Pronoun choice affects politeness and relationship:

- `saya` — neutral or formal first person singular.
- `aku` — informal/intimate first person singular.
- `kami` — exclusive “we,” excluding the addressee.
- `kita` — inclusive “we,” including the addressee.
- `Anda` — formal or neutral second person, often written.
- `kamu` — informal second person.
- `beliau` — respectful third person.

See `../pragmatics/README.md` for register and social-distance guidance.
