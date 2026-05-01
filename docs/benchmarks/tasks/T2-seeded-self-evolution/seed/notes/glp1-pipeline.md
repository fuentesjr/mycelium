# GLP-1 prescription ingestion pipeline

Pulling claims-derived prescription events for GLP-1 agonists (semaglutide, liraglutide, dulaglutide) into the analytics warehouse.

## Source

Surescripts feed lands in S3 hourly. Roughly 40-60M events/day across all drug classes; we filter to GLP-1 NDCs at the Kafka producer.

## Topology

```
S3 (hourly Surescripts drops)
  -> Kafka topic: rx.events.glp1 (partitioned by patient_id_hash, 24 partitions)
  -> Flink job: rx-normalizer (1.16, k8s operator)
  -> Postgres: rx.glp1_events (range-partitioned by fill_month)
```

## Schema sketch

```sql
CREATE TABLE rx.glp1_events (
  rx_id            BIGINT       NOT NULL,
  patient_id_hash  BYTEA        NOT NULL,
  drug_ndc         VARCHAR(11)  NOT NULL,
  dose_mg          NUMERIC(6,3),
  days_supply      INT,
  fill_date        DATE         NOT NULL,
  fill_month       DATE         NOT NULL,
  prescriber_npi   VARCHAR(10),
  payer_id         VARCHAR(20),
  ingested_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
  PRIMARY KEY (rx_id, fill_date)
) PARTITION BY RANGE (fill_month);
```

Partition strategy: monthly partitions, 13 months hot + roll older to cold storage on the 1st.

## Open

- Dedup. `rx_id` should be unique per fill but Surescripts re-emits on corrections; we need (rx_id, fill_date, payer_id) as the dedup key in the Flink job before insert.
- Backfill. Initial 24-month load is ~3.6B rows; need to size the COPY workers vs replication lag.
- Late-arriving events. Surescripts has corrected events landing weeks after fill_date — partition strategy holds, but the Flink watermark needs a 14-day lateness allowance.

## Drug NDC list

See `notes/semaglutide.md` for semaglutide NDCs. Dulaglutide and liraglutide TBD — pull from FDA NDC directory.
