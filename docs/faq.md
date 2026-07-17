# Mycelium FAQ

Quick answers for people adopting or operating Mycelium with pi coding agents.

## What is Mycelium?

Mycelium is persistent memory for pi coding agents: a local journal folder, safe CLI mutations, and a searchable JSONL activity log. The `pi-mycelium` extension installs the prompt guidance, starter journal template, lifecycle logging, and bundled Go CLI engine.

## Who is it for?

Teams using pi agents for coding, research, or operations work where context must survive across sessions and concurrent agents. It is a good fit when you want agent memory in human-readable files that can be audited with normal tools.

## Does it lock me into a model or harness?

Mycelium does not lock you into a model: benchmark coverage can include multiple frontier models running inside pi. It does intentionally support only the pi coding-agent harness. The binary may be useful from a shell for diagnostics or development, but non-pi harness integrations are unsupported.

## How is it different from a vector store or MCP memory server?

Mycelium is file-based, not retrieval-service-based. The agent navigates with `ls`, `grep`, and `read`; every byte is visible to humans; and no daemon, network service, embeddings index, or database is required.

## How do I install it?

```bash
pi install npm:pi-mycelium        # global journal
pi install npm:pi-mycelium -l     # project-local journal
```

The npm package resolves the platform-matching bundled CLI package. Source builds are for development, diagnostics, and advanced operation, not a supported alternate harness path.

## What does the pi extension configure?

On session start it resolves the bundled binary, sets `MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, and `MYCELIUM_SESSION_ID`, bootstraps `MYCELIUM_MEMORY.md` if needed, and appends a session-boundary activity entry. Before the agent starts it injects concise operating guidance.

## Can I use it outside pi?

Direct CLI invocations are useful for inspecting a journal, debugging packaging, or developing the engine. Set `MYCELIUM_MOUNT` and run `mycelium <subcommand>`. That does not make Claude Code, Codex, Hermes, custom scripts, or third-party adapters supported Mycelium integrations.

## Does it need a daemon, database, or network access?

No. Each CLI invocation opens the local POSIX journal, performs one operation, writes any activity entry, and exits.

## Can a misbehaving agent escape the mount?

Mycelium is not a sandbox. It protects the integrity of mutations made through `mycelium` inside the journal, including reserved `_` paths, CAS, and durable logging. pi and the operating system control broader shell permissions.

## What happens if the process crashes during a write?

Content mutations are atomic and fsynced before the activity entry is appended and fsynced. If the process dies before content commit, the file is unchanged. If power is lost after content commit but before log append, the content may exist without the matching log line; if the append fails in-process after commit, the command exits non-zero.

## What happens when two agents write the same file?

Conditional writes with `--expected-version` prevent silent loss. One writer succeeds and returns a new version; the stale writer exits 64 with a JSON conflict envelope. Recovery is re-read with `mycelium read <path> --format json`, merge, and retry with the fresh version.

## What filesystems are supported?

Use a local POSIX filesystem on macOS or Linux. Mycelium relies on `flock`, atomic rename, `O_APPEND`, and `fsync`. iCloud, Dropbox, OneDrive, NFS, SMB, FUSE, and Windows are outside the current guarantee set.

## How do I audit activity?

Read `_activity/YYYY/MM/DD/<agent_id>.jsonl` with `cat`, `tail -f`, `grep`, or `mycelium grep`. The pi extension records session boundaries, `session_shutdown`, and `compaction`; the CLI records `write`, `edit`, `rm`, `mv`, and explicit `log` entries.

## What should agents record for future reviewers?

Use three layers: note content for per-note reasoning, `--rationale` for why a specific mutation or signal happened, and `MYCELIUM_MEMORY.md` for durable conventions and lessons. Use `mycelium log decision` or `mycelium log agent_note` for point-in-time signals that should remain grepable history.

## Should I commit a journal to git or jj?

You can. A journal is plain files plus JSONL logs. The tradeoff is log growth and noisy diffs. For now, commits are manual; future workflow integration is planned separately.

## How do I move a journal between machines?

Stop pi, archive or copy the directory, then place it at the selected install
scope's journal path: `~/.pi/agent/extensions/pi-mycelium/journal/` globally or
`<repo>/.pi/pi-mycelium/journal/` for a project-local install. The extension
selects that path and overwrites `MYCELIUM_MOUNT` at session start. Direct CLI
diagnostics may point `MYCELIUM_MOUNT` at an arbitrary copied journal. There is
no database migration.

## How big does the activity log get?

It grows with successful mutations and explicit signals. There is no built-in retention policy yet. Manual archival of older `_activity/YYYY/` subtrees is possible when you no longer need those logs online.

## Is Mycelium production-ready?

It is early-access pre-1.0. The core storage and mutation behavior has tests, and the supported integration is pi-only. API details and activity wording may still change before 1.0, while journal compatibility remains a transition constraint.

## Are benchmarks available?

The Phase 1 benchmark rubric is in [`benchmarks/phase-1.md`](benchmarks/phase-1.md). It evaluates model behavior through pi; model diversity is benchmark coverage, not harness portability.

## What's on the roadmap?

See [`mycelium-phases.md`](mycelium-phases.md). The roadmap now focuses on pi installation quality, pi session continuity, LocalFS correctness, diagnostics, and later workflow integrations such as git/jj.
