# Semaglutide NDC notes

NDC codes and brand mapping for semaglutide products in the GLP-1 ingestion pipeline.

## Brand vs indication

Semaglutide is sold under three brand names depending on indication and route:

- **Ozempic** — subcutaneous, weekly. Indicated for type 2 diabetes.
- **Wegovy** — subcutaneous, weekly. Indicated for chronic weight management. Higher max dose (2.4 mg) than Ozempic (2.0 mg).
- **Rybelsus** — oral, daily. Indicated for type 2 diabetes.

Same molecule. The pipeline should keep them separate via `drug_ndc` so analytics can distinguish indication-by-NDC.

## NDC list (abridged — confirm against FDA NDC directory before locking)

Ozempic 0.25/0.5 mg pen: 0169-4181-13
Ozempic 1 mg pen:        0169-4503-13
Ozempic 2 mg pen:        0169-4772-12
Wegovy 0.25 mg pen:      0169-4525-15
Wegovy 0.5 mg pen:       0169-4530-15
Wegovy 1 mg pen:         0169-4543-15
Wegovy 1.7 mg pen:       0169-4621-15
Wegovy 2.4 mg pen:       0169-4634-15
Rybelsus 3 mg tablet:    0169-4301-30
Rybelsus 7 mg tablet:    0169-4302-30
Rybelsus 14 mg tablet:   0169-4303-30

(Pulled these from a Surescripts sample; double-check against the NDC directory before using in the producer filter.)

## Dose extraction

For Ozempic/Wegovy pens the NDC encodes the dose strength and it's safer to derive `dose_mg` from a NDC→dose lookup than from the prescription `dose_qty` field, which is sometimes encoded as units (clicks) rather than mg.

For Rybelsus the tablet strength is the per-dose mg.
