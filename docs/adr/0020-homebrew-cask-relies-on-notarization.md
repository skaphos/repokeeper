# ADR-0020: Homebrew cask relies on notarization, not a quarantine hook

**Status:** Accepted
**Date:** 2026-10-11
**Author:** Shawn Stratton
**Supersedes:** the Gatekeeper decision in [ADR-0007](./0007-release-binaries-and-homebrew.md) §2

## Context

ADR-0007 added a cask `postflight` hook that runs `xattr -dr com.apple.quarantine` on the staged
binary, because the darwin binaries were unsigned and Gatekeeper refused to run them. It named the
long-term fix: "notarize the binary with an Apple Developer ID so Gatekeeper accepts it without any
workaround".

That fix has shipped. GoReleaser signs the darwin binaries with a Developer ID Application
certificate (hardened runtime, secure timestamp) and notarizes them through quill. On the installed
`v2.0.1` binary, `spctl -a -t install` reports `source=Notarized Developer ID`.

Homebrew now deprecates the block-form `postflight` stanza in favor of `postflight_steps`. Every
`brew` command that loads the tap prints `Calling postflight is deprecated!` and asks users to
report it to `skaphos/homebrew-tools`.

The distribution-channels spec (`specs/001-distribution-channels/spec.md`) kept the hook and left
removal as a separate decision. Its reason was that a notarization ticket cannot be stapled to a
bare Mach-O inside a tarball, so Gatekeeper checks the ticket online on first run.

## Decision

Remove the quarantine hook from the cask. Do not port it to `postflight_steps`.

- The binary passes Gatekeeper on its own, so the hook only bypasses a check the binary already
  passes.
- The online-ticket concern covers a narrow case: a first run while offline, right after a `brew
  install` that itself needed the network. Gatekeeper caches the result after the first successful
  check.
- Porting the hook to `postflight_steps` would keep a security bypass we no longer need. It would
  also require a Homebrew version recent enough to understand the new stanza.

Because nothing in the cask covers for a missing signature anymore, the release workflow now fails
before GoReleaser runs if any `MACOS_*` secret is unset. Previously GoReleaser skipped notarization
without failing the release. This follows the same rule as the Homebrew tap credential check: a
release that half-lands must fail, not warn.

## Consequences

- The rendered cask has no `postflight` block, and the deprecation warning stops once the next
  release overwrites `Casks/repokeeper.rb` in the tap.
- macOS users keep the `com.apple.quarantine` xattr on the installed binary, and Gatekeeper checks
  it as a notarized Developer ID binary.
- A first run with no network access straight after install may be refused until the machine can
  reach Apple's notarization service.
- The Developer ID certificate's expiry date is now a release dependency. Releases stop if signing
  fails, and they can no longer fall back to unsigned binaries. Renew the certificate and rotate
  `MACOS_SIGN_P12` / `MACOS_SIGN_PASSWORD` before it expires. Binaries signed earlier stay valid
  because their signatures carry a secure timestamp.
- Builds without the secrets (local snapshots, forks) still skip notarization through the
  `enabled` template in `.goreleaser.yaml`. They do not publish the cask.
