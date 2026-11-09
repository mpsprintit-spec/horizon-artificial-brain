# Horizon — Standing Research & Engineering Evaluation Policy

> Status: active project convention. Applies to Horizon research, implementation, audit, testing, pull requests, and Work reports. This policy describes HOW to evaluate progress, not a claim that any mechanism is already proven.

## Primary rule: evaluate each mechanism against its own objective

Do not demand proof of Artificial General Intelligence (AGI), universal intelligence, or broad generalization when the current task is to implement or test an individual neural mechanism. A successful mechanism is a meaningful intermediate result even though Horizon as a whole remains a research project.

Evaluate in this exact order:

1. **Tujuan** — Mekanisme apa yang sedang dikembangkan? Jelaskan fungsi dan batas target.
2. **Implementasi** — Apakah mekanisme sudah diimplementasikan, terintegrasi ke neural Brain yang sesuai, dan benar-benar berjalan?
3. **Pengujian** — Apakah hasil memenuhi kriteria pengujian yang telah ditetapkan untuk mekanisme itu? Cantumkan metrik, baseline/kontrol, cakupan pengujian, dan kegagalan spesifik yang relevan.
4. **Keputusan** — **Diterima**, **perlu perbaikan**, atau **ditolak**, berdasarkan bukti terhadap target mekanisme tersebut. "Diterima" bukan berarti sempurna, final, atau membuktikan AGI.
5. **Langkah berikutnya** — Apa pekerjaan pengembangan konkret selanjutnya? Prioritaskan tindakan yang memperbaiki kekurangan terukur.

## Scientific precision without goalpost-shifting

- When tests pass, say clearly **which tests passed** and **what functionality was demonstrated**. Distinguish unit/CI/build success from experimental proof of learned functionality; do not overclaim either.
- State real limitations when they change the engineering decision, but do not append a generic `AGI belum terbukti` to every successful mechanism report.
- Discuss AGI-level claims only when the task explicitly asks for a whole-system AGI/capability evaluation or when a specific claim under review requires it.
- Do not retroactively change an agreed local acceptance criterion to an undefined universal-intelligence criterion.
- Mechanisms in the canonical Brain are **evolving**, not immutable or guaranteed final. They may be refined, replaced, or superseded with better evidence and engineering results.
- Preserve provenance and known limits when promoting researched mechanisms from the experimental repository into the canonical Brain. No blanket migration of unreviewed experiments.
- For native Horizon intelligence, distinguish **coded neural substrate/learning rules** from **learned representations, neural operations, and thought patterns**. Code should not silently hard-code the cognitive answer that a learning experiment is meant to demonstrate.

## Preferred report format

| Penilaian | Pertanyaan dan bukti yang perlu ditulis |
| --- | --- |
| Tujuan | Mekanisme apa yang sedang dikembangkan? |
| Implementasi | Apakah mekanisme sudah terintegrasi dan berjalan? |
| Pengujian | Apakah hasil memenuhi kriteria yang ditetapkan? |
| Keputusan | Diterima, perlu perbaikan, atau ditolak? |
| Langkah berikutnya | Apa perkembangan konkret selanjutnya? |

**Analogi pengarah:** Kita tidak meminta setiap batu bata membuktikan bahwa ia sudah menjadi rumah; kita menguji apakah batu bata itu menjalankan fungsi yang dibutuhkan untuk membangun rumah.

This is a reporting and acceptance standard for staged, falsifiable development, not permission to ignore failed experiments or fabricate results.
