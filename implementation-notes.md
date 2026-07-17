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
