# AGENTS.md

This file provides guidance when working with code in this repository.

## Version Control

This repository uses **Jujutsu (`jj`)** as the primary version control system, co-located with git. Both `jj` and `git` commands work against the same history.

Prefer `jj` for day-to-day operations:

```bash
jj status          # Working copy status
jj log             # Commit graph
jj diff            # Uncommitted changes
jj new             # Create a new change (analogous to git checkout -b)
jj describe -m "message"  # Set the current change's description
jj squash          # Fold working copy into parent change
jj git push        # Push to git remote
```

Jujutsu does not require staging — all tracked file changes are automatically included in the current change.
Addtionally Jujutsu documentation can be accessed here: https://docs.jj-vcs.dev/latest/cli-reference/
