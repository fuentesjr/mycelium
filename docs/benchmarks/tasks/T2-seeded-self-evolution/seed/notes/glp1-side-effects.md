# Adverse-event capture for GLP-1 cohort

Notes on extending the analytics warehouse to capture adverse events (AEs) alongside prescription fills, so we can study GI-side-effect-driven discontinuation.

## Source for AE signal

Claims data, not the Surescripts feed. Two candidate signals:

1. **ICD-10 codes on encounters** — nausea (R11.0), vomiting (R11.10), diarrhea (R19.7), abdominal pain (R10.x). High false-positive rate; lots of GI complaints aren't drug-attributable.
2. **Discontinuation pattern** — patient on titration ladder for ≥8 weeks, then no fills for ≥60 days. Indirect, but doesn't require AE coding.

Plan to compute both and let analytics decide which to use per cohort.

## Schema additions

```sql
CREATE TABLE rx.ae_signals (
  patient_id_hash  BYTEA NOT NULL,
  signal_date      DATE  NOT NULL,
  signal_type      TEXT  NOT NULL,  -- 'icd10_gi' | 'discontinuation'
  signal_payload   JSONB,
  derived_from     TEXT  NOT NULL,  -- source pipeline run id
  PRIMARY KEY (patient_id_hash, signal_date, signal_type)
);
```

## Discontinuation logic

```sql
-- pseudo, needs window-function refactor
WITH last_fills AS (
  SELECT patient_id_hash, drug_ndc, MAX(fill_date) AS last_fill,
         COUNT(*) AS total_fills
  FROM rx.glp1_events
  WHERE drug_ndc IN (<semaglutide_ndcs>)
  GROUP BY patient_id_hash, drug_ndc
)
SELECT * FROM last_fills
WHERE total_fills >= 8
  AND last_fill < CURRENT_DATE - INTERVAL '60 days';
```

Run as a daily job; insert into `rx.ae_signals`.

## Open

- Cardiovascular AEs are a separate workstream (not in this scope).
- Pancreatitis signal — historically a concern for GLP-1 agonists; ICD-10 K85.x — add to the GI signal extractor in a later iteration.
- Discontinuation logic doesn't distinguish drug-switching (semaglutide → tirzepatide) from outright stopping. Need to flag switches separately by checking for a fill of the other drug class within the 60-day window.
