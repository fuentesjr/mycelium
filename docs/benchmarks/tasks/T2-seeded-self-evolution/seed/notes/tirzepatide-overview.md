# Tirzepatide notes

Tirzepatide is a dual GIP/GLP-1 receptor agonist — mechanically distinct from pure GLP-1 agonists like semaglutide. We're including it in the "GLP-1 pipeline" by convention because the analytics audience treats them as the same therapeutic category, even though strictly speaking it's a different drug class.

## Brand mapping

- **Mounjaro** — type 2 diabetes indication
- **Zepbound** — chronic weight management indication

Same molecule, separate NDCs.

## Dose schedule

Different from semaglutide — the dose ladder is:

| Week  | Dose (mg/wk) |
|-------|--------------|
| 1–4   | 2.5          |
| 5–8   | 5.0          |
| 9–12  | 7.5          |
| 13–16 | 10.0         |
| 17–20 | 12.5         |
| 21+   | 15.0 (max)   |

The titration view in the pipeline needs a per-drug dose ladder; can't reuse the semaglutide schedule.

## NDC TODO

Need to pull the Mounjaro and Zepbound NDC list from the FDA directory and add to the producer filter. Tracking as a separate task — for now the pipeline only ingests semaglutide.

## Naming caveat

Some external documents use "GLP-1" loosely to include tirzepatide. When writing reports, distinguish "pure GLP-1 agonists" (semaglutide, liraglutide, dulaglutide) from "GLP-1-class" (which includes tirzepatide as a dual agonist). The product team is OK with the looser usage internally; be precise in anything that goes to clinical reviewers.
