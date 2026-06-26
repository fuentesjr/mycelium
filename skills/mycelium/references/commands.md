# Command Reference

## Environment

- `MYCELIUM_MOUNT` is required and points at the memory store.
- `MYCELIUM_AGENT_ID` defaults to `agent` when unset.
- `MYCELIUM_SESSION_ID` defaults to an auto-generated per-process id when unset.

Check the local setup with:

```sh
./scripts/doctor
```

Run it from the skill directory, or call it by absolute path from wherever the
skill is installed.

## Everyday Commands

```sh
mycelium read <path> [--format text|json]
mycelium write <path> [--expected-version SHA] [--rationale STR]
mycelium edit <path> --old STR --new STR [--expected-version SHA] [--rationale STR]
mycelium ls [pattern] [--recursive]
mycelium grep --pattern STR [--path PATH] [--regex] [--format text|json] [--limit N]
```

`read --format json` returns `path`, `version`, and `content` in one envelope.
Use that version for later CAS-safe writes.

## Occasional Commands

```sh
mycelium rm <path> [--expected-version SHA] [--rationale STR]
mycelium mv <src> <dst> [--expected-version SHA] [--rationale STR]
```

`mv` fails if the destination already exists. Read the destination before
deciding whether to remove it or choose a new path.

## Metadata Command

```sh
mycelium log <op> [--path PATH] [--payload-json STR | --stdin] [--rationale STR]
```

Use `log` for point-in-time signals such as `decision` or `agent_note`.
Do not use it as a substitute for writing durable content or updating
`MYCELIUM_MEMORY.md`.

## Exit Codes

- `0`: success.
- `1`: generic failure, including not found and post-commit log append failure.
- `2`: usage error.
- `64`: CAS conflict or `mv` destination exists; stderr is JSON.
- `65`: protocol violation such as reserved `_` path or oversize rationale.
