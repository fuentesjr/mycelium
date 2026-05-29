# Mycelium Skill + Adapter Split

## Summary

Repackage the reusable agent-facing parts of `pi-mycelium` as a cross-compatible `mycelium` Agent Skill, while leaving `pi-mycelium` as the concrete adapter for pi.dev lifecycle hooks and environment setup.

This matches the repo's current architecture: Mycelium is shell-first (L0), while adapters provide L1-L3 observability and lifecycle integration where hooks exist. Claude's current skill docs describe skills as filesystem `SKILL.md` packages with optional resources, loaded on demand from user/project skill directories: https://code.claude.com/docs/en/agent-sdk/skills

## Key Changes

- Add `skills/mycelium/` as the canonical portable Agent Skill source.
- The skill should include:
  - `SKILL.md` with concise trigger metadata and operational workflow.
  - `references/` for command reference, CAS/conflict recovery, activity events, and memory guidance.
  - `scripts/doctor` or `scripts/setup` for hybrid setup: check `mycelium` on `PATH`, verify `MYCELIUM_MOUNT`, suggest or install a release binary when absent.
- Keep `extensions/pi-mycelium/` as the pi.dev adapter.
  - It still sets `MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, `MYCELIUM_SESSION_ID`.
  - After Stage 5, it records session boundaries, compaction, and deduped context checkpoints only (no turn or tool events).
  - Its system prompt should stay aligned with the skill guidance so the two do not drift.
- Add a lightweight non-pi shell adapter later, not as part of this skill, for L1 session logging and env setup in generic harnesses.

## Public Interfaces

- No change to the `mycelium` CLI contract.
- New installable skill package name: `mycelium`.
- New supported install targets:
  - Codex-style skill directories.
  - Claude-style `.claude/skills/mycelium/SKILL.md` or `~/.claude/skills/mycelium/SKILL.md`.
- New helper script interface:
  - `scripts/doctor`: read-only validation of binary, mount, env, and basic command availability.
  - `scripts/setup`: optional hybrid setup path; prefer existing `mycelium`, otherwise guide/download/install release binary with user approval.

## Test Plan

- Validate skill structure with available skill validation tooling.
- Test trigger behavior with prompts like:
  - "Use Mycelium memory for this project."
  - "Resume work from the persistent Mycelium store."
  - "Record a durable convention in Mycelium."
- Run existing Go tests and pi extension tests to ensure no regression.
- Manually verify three modes:
  - pi.dev extension still reaches L3 behavior.
  - generic shell/Codex-style use reaches L0 with the skill and CLI.
  - missing binary or missing `MYCELIUM_MOUNT` produces clear setup guidance.

## Assumptions

- Preferred architecture is skill plus adapters.
- First skill target is cross-compatible `SKILL.md`, not Codex-only or Claude-only.
- First binary distribution path is hybrid setup, not fully bundled binaries.
- The skill cannot replace harness lifecycle hooks; it can teach the agent what to do, but adapters remain responsible for automatic env setup and telemetry.
