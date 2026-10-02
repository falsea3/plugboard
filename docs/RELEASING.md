# Releasing

Maintainer notes. Contributors don't need any of this to build or test Plugboard.

A release is a `v*` tag. Pushing one runs
[`.github/workflows/release.yml`](../.github/workflows/release.yml), which builds
macOS (Apple Silicon and Intel, one download each), Windows (x64 and Arm64) and
Linux (x64), signs the files the in-app updater installs, and publishes the
GitHub release with `latest.json`.

## Once: update signing

The updater installs a file only if it matches the SHA-256 in `latest.json` *and*
carries a [minisign](https://jedisct1.github.io/minisign/) signature from the key
built into the app, whose trusted comment names that version and file. Make the
key on your own machine, never on CI:

```bash
brew install minisign
make update-keygen    # writes update-signing-key and update-signing-key.pub (both git-ignored)
```

Then, in the GitHub repository under **Settings → Secrets and variables → Actions**:

| Kind | Name | Value |
| --- | --- | --- |
| Variable | `UPDATE_PUBLIC_KEY` | the second line of `update-signing-key.pub` |
| Secret | `UPDATE_SIGNING_KEY` | the whole of `update-signing-key` |

Keep a copy of `update-signing-key` somewhere safe outside the laptop. If it is
lost, installed copies can never update again; if it leaks, make a new pair and
ship one release signed with the **old** key that carries the **new** public key.

Release builds fail without the public key, so no published build ends up unable
to update. Local builds (`make build`, `make dev`) have no key and never update.

## Cutting a release

1. Move the notes under `## [Unreleased]` in [CHANGELOG.md](../CHANGELOG.md) to
   `## [x.y.z] - YYYY-MM-DD`, leave an empty `[Unreleased]` above it, and commit.
   That section becomes the release notes and the in-app *What's new*; the
   workflow refuses a version without one.
2. Run `make release` (next patch), `make release-minor`, `make release-major`
   or `make release v=x.y.z`. It checks the changelog, shows the notes, tags and
   pushes.
3. Watch the run under **Actions**, then go through the checklist below.

## After the workflow finishes

Automation checks the tests, that each build starts and reports the tagged
version, and that the signing key matches the public key. It can't check the
installers on real machines or the update from the version people have:

- [ ] The release has `latest.json`, `SHA256SUMS.txt` and a `.minisig` next to each file `latest.json` lists.
- [ ] macOS: the `.dmg` installs, the app opens (note the Gatekeeper prompt while it isn't notarized), About shows the new version.
- [ ] Windows: the installer completes and the app opens.
- [ ] Linux: the AppImage runs.
- [ ] **Update path**: a copy of the previous release finds this one at launch, installs it, and comes back on the new version after *Restart now*.
