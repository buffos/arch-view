# Releasing Arch View

Arch View uses semantic application releases. The release version belongs to
the host-and-analyzer distribution; model, protocol, analyzer, configuration,
and quality contracts remain independently versioned.

## Release identity

Release tags use `vMAJOR.MINOR.PATCH`, with optional prerelease or build
metadata. The first application release was `v0.1.0`. The current application
release is `v0.2.0`.

- Patch releases fix defects without changing the public contract.
- Minor releases add backward-compatible functionality.
- Major releases change a public CLI, configuration, API, or artifact contract
  incompatibly.

Every release tree contains `release.json`, which records the application
version, source commit, build timestamp, build ID, target platform, host
checksum, and analyzer-index checksum. `analyzers/index.json` remains the
source of truth for analyzer package versions and integrity.

## Local release assembly

The release command accepts controlled metadata explicitly:

```text
make release VERSION=0.2.0 COMMIT=<commit> BUILD_DATE=<utc-timestamp> BUILD_ID=<build-id>
```

The default local version is `0.2.0`; local builds use `unknown` commit/date
metadata and the `local` build ID unless overridden.

## Published releases

`master` is kept releasable. After verification and release-note preparation,
tag the exact commit and push the tag:

```text
git tag -a v0.2.0 -m "Arch View v0.2.0"
git push origin v0.2.0
```

The release workflow validates the tag, builds the supported platform matrix,
archives each complete release tree, writes checksums, and publishes the
artifacts as a GitHub Release. A tag alone is not considered released until
the workflow completes successfully and the artifacts are available.

Tags are immutable release identities. If released content needs changing,
create a new patch version rather than moving an existing tag.
