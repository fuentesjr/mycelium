# Pi-only transition progress

**Started:** 2026-07-12
**Plan:** `docs/pi-only-strategy.md`
**Status:** Complete

## Scope guardrails

- Make pi the sole supported coding-agent harness.
- Keep the Go CLI and journal format as the extension's implementation boundary.
- Preserve existing journals and tolerate historical activity operations.
- Do not redesign storage, rename packages, or add unrelated pi features.

## Progress

- [x] Strategic plan written and approved.
- [x] Inventory portable guidance, documentation claims, packaging, and tests.
- [x] Add the superseding architectural decision.
- [x] Reframe README, design, FAQ, roadmap, and benchmarks around pi.
- [x] Remove the portable Agent Skill and stale references.
- [x] Collapse generic portable activity-event documentation to the pi contract.
- [x] Align changelog, release checklist, and package description.
- [x] Run consistency searches and diagnostics.
- [x] Run Go, race, extension, and packaging verification.
- [x] Complete fresh-context review and resolve accepted findings.

## Decisions and tradeoffs

- The CLI remains a separate Go component because safe mutations, CAS, and
  durable logging earn that boundary independently of cross-harness support.
- Unsupported harness use is not intentionally blocked; it is simply not
  documented, tested, or considered part of the compatibility contract.
- Existing ADRs and changelog entries remain as history. ADR-0007 supersedes
  ADR-0002 and ADR-0006 rather than rewriting their historical decisions.
- Multiple-model benchmark coverage remains. Model diversity inside pi is not
  a multi-harness support claim.
- Platform CLI npm packages and GitHub binary archives remain because they ship
  and diagnose the Go engine used by `pi-mycelium`.
- The runtime prompt received only concise pi lifecycle wording. Durable naming,
  indexing, archiving, and activity-reading guidance lives in the starter
  `MYCELIUM_MEMORY.md` template instead of duplicating the removed skill.

## Changed areas

- Added ADR-0007, `docs/pi-only-strategy.md`, and `docs/pi-activity-events.md`.
- Reframed public docs, roadmap, benchmark wording, extension docs, and package
  description around pi-only support.
- Removed `skills/mycelium/`, the portable event specification/fixture, and its
  extension fixture test.
- Preserved the Go CLI, journal layout, unknown-event tolerance, release
  pipeline, package names, and versions.

## Verification log

- `go test ./...` — passed (`internal/mycelium`, CLI package, and T3 tooling).
- `go test -race ./internal/mycelium` — passed.
- `npm test --prefix extensions/pi-mycelium` — passed (6 files, 40 tests).
- `lsp_diagnostics` on the changed system prompt and test — no diagnostics.
- `npm pack --dry-run --prefix extensions/pi-mycelium` — failed because npm
  resolved the repository root rather than the prefixed package; no artifact
  was produced.
- `(cd extensions/pi-mycelium && npm pack --dry-run --json)` — passed; package
  contains 9 expected extension/template files and no removed portable assets.
- Consistency searches — no deleted-path references outside historical
  changelog entries or the strategy; no stale benchmark support wording; no
  malformed replacement artifacts.
- Changed-Markdown local-link check — all local targets resolve.
- Two fresh-context reviewers completed. Accepted findings were fixed: removed
  generated subagent artifacts, repaired ADR-0002's deleted references,
  enumerated exact pi session event names, renamed remaining benchmark
  "model-agnostic" criteria, and corrected T3's 3-trajectory count.
- Final `go test ./...` and `npm test --prefix extensions/pi-mycelium` reruns —
  passed (6 extension files, 40 tests).

## Residual follow-ups

- Re-evaluate standalone binary archives after the support-contract transition.
- Re-evaluate extension-to-binary identity only with evidence from pi usage.
- Global and project-local installation smoke tests require publishing/installing
  a release artifact and remain release-time verification.
