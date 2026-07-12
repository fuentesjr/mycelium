# Release Checklist

## One-time setup

- [ ] npm account with publish access to the `@fuentesjr` scope and the `pi-mycelium` package.
- [ ] Add `NPM_TOKEN` secret to the GitHub repo. Use an npm automation/granular token with publish access to all platform CLI packages and `pi-mycelium`.
- [ ] The workflow has `id-token: write` permission, so npm provenance is enabled automatically.

## Cutting a release

1. Decide the version. Update both version files to match the tag; do not bump versions for documentation-only transition work until a release is actually being cut:
   - `Makefile`: `VERSION ?= vX.Y.Z`
   - `extensions/pi-mycelium/package.json`: `"version": "X.Y.Z"`

2. Add a `CHANGELOG.md` entry. For the pi-only transition, call out the support narrowing, skill removal, event-doc replacement, and journal compatibility.

3. Run local verification:

   ```bash
   go test ./...
   go test -race ./internal/mycelium
   npm test --prefix extensions/pi-mycelium
   (cd extensions/pi-mycelium && npm pack --dry-run)
   ```

4. Confirm support-boundary consistency:
   - README and FAQ lead with `pi install npm:pi-mycelium`.
   - Direct CLI docs are framed as development, diagnostics, advanced operation, and pi's shell-invoked engine.
   - No active docs present the removed portable skill, cross-harness adapter conventions, or non-pi harnesses as supported.
   - The npm dry run includes the extension files, template, and platform CLI optional-dependency metadata, and excludes removed portable artifacts.

5. Commit and push the version bump on `main`.

6. Tag and push:

   ```bash
   git tag vX.Y.Z && git push origin vX.Y.Z
   # or with jj:
   jj bookmark create vX.Y.Z -r @ && jj git push --bookmark vX.Y.Z
   ```

7. Watch the workflow. It validates version consistency, runs tests, builds the platform CLI archives/packages, and publishes the pi extension.

8. Verify:
   - GitHub release has the binary archives attached for supported platforms.
   - npm shows the new `pi-mycelium` version and matching `@fuentesjr/mycelium-cli-*` packages.
   - A global and project-local `pi install npm:pi-mycelium` bootstrap/resume the expected journal paths.

## When things go wrong

**Version mismatch** — fix the lagging file, delete the tag, and push a corrected tag.

**Test failure on tag** — no artifacts are published. Fix on `main`, bump to a new patch version, and retag.

**npm authentication/access failure** — rotate the token to one with publish access to all required packages, then rerun or cut a clean patch release.

**npm publish partial failure** — npm may not allow clean unpublishing. Bump to the next patch, document the orphaned version in `CHANGELOG.md`, and run a clean release.

**GitHub release created but npm publish failed** — delete the GitHub release manually if needed, then bump and retry the whole release.
