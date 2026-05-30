#!/usr/bin/env bash
#
# verify-mcp.sh
#
# Sets up an isolated, reproducible environment for manual end-to-end
# verification of the RepoKeeper MCP server from Claude Code (or other clients).
#
# This directly supports closing SKA-201.
#
# Usage:
#   ./scripts/verify-mcp.sh [--keep]
#
#   --keep   Do not delete the temporary workspace after the script exits
#            (useful for capturing evidence or running the manual verification).
#
# After running, follow the printed instructions to point Claude Code at
# the generated .mcp.json and run through the verification checklist.

set -euo pipefail

KEEP_DIR=false
if [[ "${1:-}" == "--keep" ]]; then
    KEEP_DIR=true
    shift
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

TMP_DIR="$(mktemp -d -t repokeeper-mcp-verify-XXXXXX)"

if [[ "$KEEP_DIR" == "false" ]]; then
    trap 'rm -rf "$TMP_DIR"' EXIT
fi

echo "==> Creating isolated verification workspace at: $TMP_DIR"

echo "==> Creating isolated verification workspace at: $TMP_DIR"

# Create a minimal .repokeeper.yaml
cat > "$TMP_DIR/.repokeeper.yaml" << 'EOF'
roots:
  - "$TMP_DIR/repos"
defaults:
  main_branch: main
EOF

# Create some test git repositories
mkdir -p "$TMP_DIR/repos"

for name in alpha beta gamma; do
    repo_path="$TMP_DIR/repos/$name"
    mkdir -p "$repo_path"
    (cd "$repo_path" && git init -q && git checkout -q -b main)
    echo "# $name" > "$repo_path/README.md"
    (cd "$repo_path" && git add . && git commit -q -m "Initial commit for $name")
done

# Make gamma look a bit different (for status testing)
echo "dirty change" >> "$TMP_DIR/repos/gamma/README.md"

echo "==> Workspace ready."
echo ""

# Generate a ready-to-use .mcp.json for Claude Code
MCP_JSON="$TMP_DIR/.mcp.json"
cat > "$MCP_JSON" << EOF
{
  "mcpServers": {
    "repokeeper-verify": {
      "command": "repokeeper",
      "args": ["mcp", "--config", "$TMP_DIR/.repokeeper.yaml", "--log-file", "$TMP_DIR/mcp.log"]
    }
  }
}
EOF

echo "==> Generated Claude Code config:"
echo "    Copy this file or its contents:"
echo "    $MCP_JSON"
echo ""
echo "    Or use this snippet in your Claude Code settings:"
cat "$MCP_JSON"
echo ""

echo "==> Verification Environment Summary"
echo "    Workspace root:     $TMP_DIR"
echo "    Config file:        $TMP_DIR/.repokeeper.yaml"
echo "    Test repos:"
ls -1 "$TMP_DIR/repos"
echo "    Log file (after starting): $TMP_DIR/mcp.log"
echo ""

echo "==> Next Steps (for SKA-201)"
echo ""
echo "1. Point Claude Code at the MCP server using the .mcp.json above"
echo "   (either at workspace root or in your Claude settings)."
echo ""
echo "2. Start a new conversation in Claude Code with RepoKeeper context."
echo ""
echo "3. Work through the checklist below. Record pass/fail per tool."
echo ""
echo "4. When finished, post results as a comment on SKA-128 (or SKA-201)."
echo ""

cat << 'CHECKLIST'

## MCP Verification Checklist (run from Claude Code)

### Tool Discovery
- [ ] Ask Claude to list available RepoKeeper tools. Confirm exactly 14 tools appear.
- [ ] Verify readOnlyHint / destructiveHint annotations look reasonable on the tools.

### Read-only Tools
- [ ] list_repositories
- [ ] get_repository_context (try alpha, beta, gamma, and a non-existent repo)
- [ ] get_workspace_config
- [ ] build_workspace_inventory
- [ ] select_repositories (with various label/field selectors)
- [ ] get_repo_metadata, get_authoritative_paths, get_related_repositories

### Planning Tools
- [ ] plan_sync (with and without filters)
- [ ] Confirm it never mutates anything

### Mutation Tools + Safety
- [ ] scan_workspace (on the temp workspace)
- [ ] execute_sync
  - [ ] Rejects without confirm=true (or with confirm=false)
  - [ ] Succeeds with confirm=true (observe the plan execution)
- [ ] set_labels (add/remove labels on a repo)
- [ ] add_repository
- [ ] remove_repository (try tracking-only vs full delete)

### Structured Content & Errors
- [ ] Confirm list-returning tools return structuredContent as objects (not bare arrays)
- [ ] Trigger and observe good error messages for invalid inputs

### Overall
- [ ] No unexpected "expected record / received array" errors
- [ ] Claude can reason correctly using the tool outputs
- [ ] Log file at $TMP_DIR/mcp.log looks clean

CHECKLIST

if [[ "$KEEP_DIR" == "true" ]]; then
    echo ""
    echo "==> --keep was used. Workspace preserved at: $TMP_DIR"
    echo "    You can manually delete it later when finished with verification."
else
    echo ""
    echo "==> When you are done, you can safely delete: $TMP_DIR"
fi

echo ""
echo "Script complete. Good luck with the verification!"