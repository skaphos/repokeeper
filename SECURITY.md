# Security Policy

RepoKeeper inventories local repositories and provides CLI and stdio MCP
operations for inspection, registry management, synchronization, and explicitly
requested repository changes. Security-sensitive assets include working trees,
uncommitted work, repository history, workspace configuration, and credentials
available to the invoked version-control tools.

## Reporting a vulnerability

**Please do not open a public issue or pull request for a suspected
vulnerability.** Use [GitHub private vulnerability reporting](https://github.com/skaphos/repokeeper/security/advisories/new).
If that channel is unavailable or you cannot use GitHub, email
[shawn@skaphos.io](mailto:shawn@skaphos.io) with `repokeeper` in the subject.

## Threat model and reportable issues

RepoKeeper runs with the invoking user's operating-system permissions and
uses installed version-control tools. Repository names, paths, remotes,
metadata, imported manifests, and version-control output can contain
attacker-controlled data. Such data must not expand a requested operation into
unintended commands, destinations, or destructive changes.

Report suspected violations such as:

- Command or argument injection through repository metadata, paths, remote
  references, or version-control output.
- Path traversal, symlink handling, or ambiguous repository selection causing
  reads, writes, cloning, deletion, or reset of an unintended target.
- Inspection tools performing unexpected destructive operations, or mutation
  tools being represented as read-only to an MCP client.
- Bypassing an operation's required opt-in, confirmation, plan, or safety
  checks, resulting in unintended changes to a working tree or history.
- Registry or configuration writes corrupting unrelated entries or losing
  data through attacker-triggered concurrent operations.
- Credentials or private repository information leaking through logs, errors,
  output, or requests to an unintended destination.
- Exploitable resource exhaustion from untrusted repository metadata or
  version-control output.

RepoKeeper is not entirely read-only: it exposes deliberate registry and
repository mutations. The MCP server shares the user's permissions and is not
an isolation sandbox. Describe how the observed behavior exceeds the requested
operation or defeats a documented protection. These boundaries provide triage
context, not blanket exclusions: report privately when in doubt.

## What to include

Include the affected repository, version or commit, platform and relevant
configuration, reproduction steps or a minimal proof of concept, and the
impact you believe is possible. Explain what an attacker controls and which
security boundary is crossed. A suggested fix is welcome but not required.

Use synthetic data and redact credentials and private repository content.
Please report suspected issues even if you cannot reproduce them on the latest
release or are unsure whether they are vulnerabilities.

## Response and coordinated disclosure

- We aim to acknowledge reports within **7 days**. If you have not heard back,
  follow up by email with the repository name and the date of your report.
- Maintainers assess the report with the reporter, including affected versions,
  realistic exploitability, impact, and severity. A scanner alert alone does
  not establish impact, but reports do not need a complete exploit to be useful.
- We limit embargoed information to people needed for triage, remediation, and
  coordinated release. We develop and review security fixes privately and test
  that the reported issue is resolved before publishing the patch.
- We aim to release a fix or mitigation and disclose within **90 days** of the
  report. This is a coordination target, not a guaranteed fix date or an
  obligation on reporters to remain silent indefinitely. We discuss timing
  changes with the reporter; active exploitation can require earlier notice.
- We coordinate the patched release with a public security advisory describing
  impact, affected and fixed versions, and any mitigation. For vulnerabilities
  eligible for a CVE, we request an identifier through GitHub or another CVE
  Numbering Authority and include it in the advisory. Exploit details may be
  delayed to give users time to update.
- We credit reporters unless they prefer anonymity. If triage finds a regular
  bug or a hardening opportunity, we explain why and coordinate moving it to a
  public issue only after checking that doing so exposes no unresolved
  vulnerability.

## Versions and dependencies

Security fixes normally target the latest release. Report the version you use;
we assess affected versions and any need for backports during triage rather
than rejecting reports solely because they concern an older release. For
unreleased projects, include the commit you tested.

Dependency vulnerabilities are relevant when they affect this project's use of
the dependency. Include that context if known; maintainers will coordinate
with upstream as needed. Do not disclose an embargoed upstream issue publicly.

## Bug bounty

This policy does not offer a paid bug bounty.
