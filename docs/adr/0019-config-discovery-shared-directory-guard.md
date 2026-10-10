# ADR-0019: Config discovery shared-directory guard per platform

**Status:** Accepted
**Date:** 2026-10-10
**Author:** Shawn Stratton

## Context

Runtime config resolution walks upward from the working directory looking for the nearest
`.repokeeper.yaml` (`DESIGN.md` §6.2.1). #265 bounded that walk so a config planted in a shared
ancestor cannot be silently adopted:

- the walk stops at the user's home directory (inclusive);
- it never ascends into a *shared* directory: one that other local users can write into, such as
  `/tmp`;
- the candidate must be a regular file.

`isSharedDir` decides "shared" from Unix permission bits: world-writable (`o+w`) or sticky. Windows
has no Unix permission bits. Go synthesizes a mode for Windows directories that is always `0777`, so
`isSharedDir` returned true for **every** Windows directory and the walk never left the working
directory. On Windows, nearest-parent discovery has been broken since #265: running `repokeeper`
from a subdirectory of a workspace silently fell back to the global config under `%APPDATA%`. The
Windows CI job could not catch this, because `scripts/test-cover.ps1` swallowed test failures
(#364, #379).

## Decision

Keep one cross-platform rule, "do not ascend into a directory that principals broader than the
current user can create files in", and implement it per platform:

- **Unix** (`!windows`): unchanged. The directory is shared when it is world-writable or sticky.
- **Windows**: read the directory's DACL with `GetNamedSecurityInfo`. The directory is shared when
  an effective (not inherit-only) access-allowed ACE grants a **broad principal** any right that
  lets it create a file in the directory or take control of it:
  - **Broad principals:** `Everyone` (`S-1-1-0`), `Authenticated Users` (`S-1-5-11`) and
    `BUILTIN\Users` (`S-1-5-32-545`).
  - **Rights:** `FILE_ADD_FILE` / `FILE_WRITE_DATA`, `GENERIC_WRITE`, `GENERIC_ALL`, `WRITE_DAC` and
    `WRITE_OWNER`.
  - A NULL DACL grants everyone full access, so it counts as shared.
  - If the security descriptor cannot be read, or has no DACL, the directory counts as shared. The
    guard fails closed: the walk stops, and resolution falls back to the global config.

Deny ACEs are not evaluated, so a directory whose broad grant is later denied is still treated as
shared. This errs toward stopping the walk, which is the safe direction.

## Consequences

- Nearest-parent `.repokeeper.yaml` discovery works on Windows for the normal layout: a workspace
  under the user profile, whose ACLs grant only the owner, `SYSTEM` and `Administrators`.
- Typical shared locations stop the walk: `C:\Users\Public`, and folders created under a drive root
  that inherit `Authenticated Users: Modify`. The drive root `C:\` itself grants `Authenticated
  Users` only folder creation on the root (`FILE_ADD_SUBDIRECTORY`); its file-write grant is
  inherit-only, so it is not shared for this rule. A `C:\.repokeeper.yaml` requires an
  administrator to create.
- `golang.org/x/sys/windows`, already a direct dependency, is now imported on Windows builds.
- The Unix sticky-directory test is Unix-only. A Windows test builds a genuinely shared directory by
  granting `Everyone` `FILE_ADD_FILE`, and asserts the walk stops before a config planted there.
