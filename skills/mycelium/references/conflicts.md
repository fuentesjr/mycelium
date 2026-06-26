# Conflict Recovery

## Pattern

Use **re-read, merge, retry**.

1. Re-read the path:

```sh
mycelium read <path> --format json
```

2. Merge your intended change with the current content.
3. Retry with the fresh `version` token:

```sh
printf '%s' "$merged" | mycelium write <path> --expected-version sha256:<current>
```

## Operation Guidance

- `write`: combine your intended content with the current file instead of
  overwriting stale bytes.
- `edit`: relocate the old substring. If it is gone, decide whether the change
  still makes sense.
- `rm`: re-read before deleting; abort if the current content no longer merits
  removal.
- `mv destination_exists`: read the destination before choosing a new path or
  deliberately removing it.

## When To Stop

Stop and surface the issue when:

- the same path conflicts repeatedly after a sensible merge,
- current content contradicts the intended semantic change,
- a destination collision contains valuable content.

In those cases, a `mycelium log race_observed --path <path> --rationale "..."`
entry can be useful, but do not loop.
