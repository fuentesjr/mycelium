# Project review findings

Reviewed: 2026-07-12
Resolution updated: 2026-07-12

## 1. High — Symlink traversal escapes the mount

**Status: resolved.** Mycelium now rejects symlink components before read, grep scope, and all mutating filesystem operations, and skips symlink entries during `grep`/`ls` walks. The activity log append path is also checked after directory creation.

Tests added/updated: symlink rejection coverage in `path_test.go`, `read_test.go`, `write_test.go`, `edit_test.go`, `rm_test.go`, `mv_test.go`, `grep_test.go`, and `listing_test.go`.

Residual note: this is a stdlib-only hardening pass. It rejects symlink components observed before use but is not an OS-level `openat`/`O_NOFOLLOW` sandbox against hostile concurrent filesystem mutation by non-cooperating processes.

## 2. High — `MYCELIUM_AGENT_ID` permits arbitrary-path log writes

**Status: resolved.** Activity appends now validate `agent_id` before constructing/writing the daily log path. Valid IDs are empty/default or ASCII letters, digits, `.`, `_`, and `-`, excluding `.`/`..` and overlong values. Invalid IDs fail the log operation and do not create files outside the mount.

Tests added/updated: `TestValidateAgentID`, `TestLogRejectsAgentIDPathTraversal`, and `TestAppendActivityLineDurableRejectsInvalidAgentID`.

## 3. Medium — `grep --path <file>` silently returns no matches

**Status: resolved.** `grep` now skips the walk root only when the root is a directory; file-scoped searches scan the file directly.

Tests added/updated: `TestGrepPathScopeCanBeFile` covers text and JSON file-scope output.

## 4. Low — Exact-limit grep results incorrectly report truncation

**Status: resolved.** `grep` now scans for one additional match before reporting truncation. Exact-limit results are no longer marked truncated, while over-limit cases remain capped and marked.

Tests added/updated: `TestGrepExactLimitNotTruncated`; existing over-limit tests continue to verify true truncation.

## Verification

Verified on 2026-07-12:

- Focused path tests — passed (`16.842s`)
- Focused identity/log tests — passed (`2.009s`)
- Focused grep/listing tests — passed (`0.407s`)
- Focused mutation tests — passed (`2.567s`)
- `go test ./...` — passed (`internal/mycelium` in `27.590s`)
- `go test -race ./internal/mycelium` — passed (`51.238s`)
- `npm test --prefix extensions/pi-mycelium` — passed (`7` files, `41` tests)
