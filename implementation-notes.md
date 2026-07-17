# Implementation Notes

## Scope

- Make the default local test path fast without weakening production durability.
- Preserve exhaustive property and race coverage behind explicit targets and CI gates.
- Do not change property counts, concurrency coverage, tar coverage, or production code.

## Decisions

- `testing.Short()` skips only the three 50-case deterministic property tests.
- `make test` aliases the fast all-language suite; `test-full` and `test-race` are explicit.
- PRs and `main` pushes run fast feedback and exhaustive Go verification in parallel.
- Releases retain full Go coverage and gain the short race gate before publication.

## Verification

- Pre-change: `make test` selected full Go coverage; `-short` still ran property cases.
- `make test`: passed in 8.55 seconds; Go fast path 4.78 seconds, T3 and 40 TypeScript tests passed.
- `make test-full`: passed; exhaustive Go package 39.95 seconds and 40 TypeScript tests passed.
- `make test-race`: passed; race-tested Go package 22.47 seconds.
- Targeted short run confirmed all three property tests report skipped.
- Both GitHub workflow files parse as valid YAML; final diff checks passed.

## Documentation review completion (2026-07-17)

### Scope

- Finish the delegated documentation findings after the test-suite split.
- Repair any runtime/documentation mismatches discovered during review.
- Make T1/T2 benchmark selection and grading reproducible before first runs.

### Decisions

- Keep the runtime fixes separate from the documentation reconciliation commit.
- Treat T1's answer key as private operator/grader input held outside the
  public repository and the evaluated model's filesystem namespace; provision
  it only in a separate post-run grading environment and pin it per campaign by
  checksum instead of committing the truth set.
- Preserve historical ADR decisions while adding prominent supersession and
  current-implementation corrections where examples could mislead operators.
- Document bootstrap and lifecycle events as best-effort because the extension
  intentionally allows pi to continue when those writes fail.
- Preserve the well-known `MYCELIUM_MEMORY.md` path; agents may replace its
  contents, but deletion triggers a reseed attempt on the next session.
- Make version-pinned npm registrations project-local so the reproducible
  benchmark install command resolves the documented mount.

### Verification

- The T1 answer-key contract requires primary sources, check dates, external
  storage, and a recorded checksum; the researched truth set is intentionally
  absent from the public repository.
- A focused red-green test reproduced and fixed version-pinned project-scope
  detection; all 8 config tests pass.
- `pi --help` on pi 0.80.10 confirms the benchmark isolation, exact-model, and
  optional model-search flags used by the harnesses.
- Local-link validation checked all 36 first-party Markdown files; every
  relative target exists, and all 24 bash/sh fences parse successfully.
- `make test`: fast Go suite and T3 detectors passed; TypeScript typecheck and
  all 41 extension tests passed.
- `make test-full`: exhaustive Go suite and T3 detectors passed; TypeScript
  typecheck and all 40 extension tests passed.
- Final agenticons re-review, stale-contract, and whitespace/diff checks passed.
