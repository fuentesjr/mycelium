# GLP-1 ingestion pipeline notes

Working notes on the GLP-1 prescription event pipeline. Source is the Surescripts feed; sink is `rx.glp1_events` in the analytics Postgres.

## Components

- Hourly S3 landing (Surescripts)
- Kafka topic `rx.events.glp1`, 24 partitions, key = patient_id_hash
- Flink normalizer (rx-normalizer) on 1.16
- Postgres sink, monthly range partitions

## Dedup key

After looking at a sample of Surescripts corrections, the right dedup tuple is `(rx_id, fill_date, payer_id)` — same rx can be re-emitted with a corrected payer or with quantity adjustments and we want the latest version, not multiple rows.

Implement as a Flink ProcessFunction with state-TTL of 30 days, keyed by the tuple. Emit downstream only when the incoming event's `ingested_at` is newer than the cached version.

## Throughput sizing

40-60M events/day filtered to GLP-1 NDCs gives roughly 800K-1.2M events/day for our subset. Steady-state Kafka throughput is well under 100 msg/sec — the bottleneck is the backfill, not steady state.

## Backfill plan

24 months of history, ~3.6B raw events filtering to ~70M GLP-1. Two-stage:

1. Bulk-load raw NDC-filtered rows directly into a staging table via parallel COPY (8 workers, one per `fill_month` partition window).
2. Run the dedup logic in batch over the staging table, insert into `rx.glp1_events`.

Avoid pushing backfill through Kafka — at 70M events the Flink job would take days and we'd risk replication lag on the sink Postgres.

## TODO

- Confirm the NDC list for liraglutide and dulaglutide (semaglutide list lives elsewhere)
- Wire alerting on Flink checkpoint failure
- Decide retention on the Kafka topic (7 days vs 30)
