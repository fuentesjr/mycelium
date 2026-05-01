# GLP-1 dose-titration schema followup

Picking up the pipeline thread — the rx_events schema captures *fills* but not the *titration arc* (the typical dose escalation pattern over weeks 1–16). Analysts asked for a derived view that makes titration adherence queryable.

## Titration model

Standard semaglutide initiation (subq, weight management):

| Week | Dose (mg/wk) |
|------|--------------|
| 1–4  | 0.25         |
| 5–8  | 0.5          |
| 9–12 | 1.0          |
| 13–16| 1.7          |
| 17+  | 2.4 (maintenance) |

Tirzepatide has its own escalation schedule, mostly parallel structure but different mg values — handle per-drug.

## Derived view

```sql
CREATE MATERIALIZED VIEW rx.glp1_titration AS
SELECT
  patient_id_hash,
  drug_ndc,
  fill_date,
  dose_mg,
  ROW_NUMBER() OVER (
    PARTITION BY patient_id_hash, drug_ndc
    ORDER BY fill_date
  ) AS fill_seq,
  fill_date - FIRST_VALUE(fill_date) OVER (
    PARTITION BY patient_id_hash, drug_ndc
    ORDER BY fill_date
  ) AS days_since_first
FROM rx.glp1_events;
```

`fill_seq` + `days_since_first` lets analysts compare actual vs expected escalation curve.

Refresh nightly, after the day's pipeline run completes. Refresh time on a 70M-row source is 4-7 minutes on the analytics box.

## Adherence score TBD

A simple "what % of expected fills did this patient have within ±7 days of the expected date" metric would be useful but requires patient-level expected schedules — defer until product confirms what cohort definition they want.

## Cross-ref

The base `rx.glp1_events` table is documented in the pipeline notes.
