# Release Checklist

## One-time setup

- [ ] npm account with publish access to the `@fuentesjr` scope and the `pi-mycelium` package.
- [ ] Add `NPM_TOKEN` secret to the GitHub repo: Settings → Secrets and variables → Actions → New repository secret. Use an **Automation** type token (works with 2FA-enabled accounts).
- [ ] The workflow already has `id-token: write` permission, so npm provenance is enabled automatically — no extra steps needed.

## Cutting a release

1. Decide the version (semver). Update both version files to match — they must agree with the tag or the workflow will fail:
   - `Makefile`: `VERSION ?= vX.Y.Z`
   - `extensions/pi-mycelium/package.json`: `"version": "X.Y.Z"` (no leading `v`)

2. Add a `CHANGELOG.md` entry:
   ```
   ## [X.Y.Z] - YYYY-MM-DD
   - What changed and why.
   ```

3. Commit and push the version bump on `main`.

4. Tag and push:
   ```bash
   # With git:
   git tag vX.Y.Z && git push origin vX.Y.Z

   # With jj:
   jj bookmark create vX.Y.Z -r @ && jj git push --bookmark vX.Y.Z
   ```

5. Watch the workflow at `https://github.com/fuentesjr/mycelium/actions`. It validates version consistency, runs tests, builds, and publishes — no manual steps needed if it stays green.

6. Verify:
   - GitHub release: `https://github.com/fuentesjr/mycelium/releases/tag/vX.Y.Z` — four binary tarballs should be attached.
   - npm: `https://www.npmjs.com/package/pi-mycelium` — new version should be live.

## When things go wrong

**Version mismatch** — the workflow fails the consistency check before anything is published. Fix the version in the lagging file (`Makefile` or `package.json`), commit, delete the tag, and push a new one:
```bash
git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z
# fix the file, commit, then:
git tag vX.Y.Z && git push origin vX.Y.Z
```

**Test failure on tag** — no artifacts are published. Fix the bug on `main`, bump to a new patch version, retag.

**npm publish partial failure** — some platform packages published but others (or the meta package) did not. npm does not allow unpublishing within 72 hours of publish. Do not attempt to unpublish. Instead: bump to the next patch version (e.g. `0.1.0 → 0.1.1`), document the orphaned version in `CHANGELOG.md`, and run a clean release from scratch.

**GitHub release created but npm publish failed** — delete the GitHub release manually (GitHub UI → Edit → Delete release, keep the tag or delete and recreate it), then bump the version and retry the whole release.
