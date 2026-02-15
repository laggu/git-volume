#!/bin/bash
set -eo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
TEST_DIR=$(mktemp -d)
GV_BIN="${PROJECT_ROOT}/git-volume"
FAILED=0

# Compile git-volume if not present
if [[ ! -f "$GV_BIN" ]]; then
    echo "🔨 Building git-volume..."
    (cd "$PROJECT_ROOT" && go build -o "$GV_BIN" .)
fi

# Helper Functions
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

log() {
    echo "[$1] $2"
}

pass() {
    echo "✅ PASS: $1"
}

fail() {
    echo "❌ FAIL: $1"
    FAILED=1
}

# Setup Test Environment
mkdir -p "$TEST_DIR/global"
export HOME="$TEST_DIR" # Mock HOME for global config

log "SETUP" "Test directory: $TEST_DIR"

# -----------------------------------------------------------------------------
# Test: init
# -----------------------------------------------------------------------------
log "TEST" "Testing 'init' command..."
cd "$TEST_DIR"
mkdir project
cd project

# Initialize git repository (required for git-volume)
git init -q

"$GV_BIN" init -q
if [[ -d "$TEST_DIR/.git-volume" && -f "git-volume.yaml" ]]; then
    pass "init created necessary files"
else
    fail "init failed to create files"
fi

# -----------------------------------------------------------------------------
# Test: sync & status (Mixed Mode)
# -----------------------------------------------------------------------------
log "TEST" "Testing 'sync' & 'status' (Link + Copy)..."

# Create source file
echo "SECRET_DATA" > source.txt

# Create config with both link and copy
cat > git-volume.yaml <<EOF
volumes:
  - mount: "source.txt:target_link.txt"
    mode: link
  - mount: "source.txt:target_copy.txt"
    mode: copy
EOF

# Commit config so it exists in worktrees
git add git-volume.yaml >/dev/null 2>&1
git commit -m "Add git-volume config" >/dev/null 2>&1

# Run Sync
"$GV_BIN" sync

# Verify Link
if [[ -L "target_link.txt" ]]; then
    CONTENT=$(cat target_link.txt)
    if [[ "$CONTENT" == "SECRET_DATA" ]]; then
        pass "sync created symlink correctly"
    else
        fail "symlink content mismatch"
    fi
else
    fail "sync failed to create symlink"
fi

# Verify Copy
if [[ -f "target_copy.txt" && ! -L "target_copy.txt" ]]; then
    CONTENT=$(cat target_copy.txt)
    if [[ "$CONTENT" == "SECRET_DATA" ]]; then
        pass "sync created copy correctly"
    else
        fail "copy content mismatch"
    fi
else
    fail "sync failed to create copy"
fi

# Run Status
OUTPUT="$("$GV_BIN" status)"
if grep -q "target_link.txt.*OK" <<< "$OUTPUT" && \
   grep -q "target_copy.txt.*OK" <<< "$OUTPUT"; then
    pass "status confirmed volumes are mounted"
else
    fail "status output incorrect or missing volumes"
    echo "Output: $OUTPUT"
fi

# -----------------------------------------------------------------------------
# Test: unsync
# -----------------------------------------------------------------------------
log "TEST" "Testing 'unsync'..."

"$GV_BIN" unsync

if [[ ! -e "target_link.txt" ]]; then
    pass "unsync removed symlink"
else
    fail "unsync failed to remove symlink"
fi

if [[ ! -e "target_copy.txt" ]]; then
    pass "unsync removed copy"
else
    fail "unsync failed to remove copy"
fi

# -----------------------------------------------------------------------------
# Test: unsync safety (Modified Copy)
# -----------------------------------------------------------------------------
log "TEST" "Testing 'unsync' safety check..."

# Re-sync
"$GV_BIN" sync

# Modify copied file
echo "MODIFIED_DATA" > target_copy.txt

# Unsync
"$GV_BIN" unsync

if [[ -f "target_copy.txt" ]]; then
    CONTENT=$(cat target_copy.txt)
    if [[ "$CONTENT" == "MODIFIED_DATA" ]]; then
        pass "unsync preserved modified file"
    else
        fail "unsync modified file content changed"
    fi
else
    fail "unsync deleted modified file"
fi

# Clean up modified file for next tests if needed (not needed here but good practice)
rm -f target_copy.txt

# -----------------------------------------------------------------------------
# Test: Worktree Inheritance
# -----------------------------------------------------------------------------
log "TEST" "Testing 'Worktree Inheritance'..."

# Create a worktree WITHOUT git-volume.yaml inside it
git worktree add ../feat-1 -b feat-1 >/dev/null 2>&1
pushd ../feat-1 >/dev/null

# Since we committed git-volume.yaml, it exists here. Remove it to test inheritance.
rm git-volume.yaml

# Run sync in worktree (expecting inheritance)
"$GV_BIN" sync >/dev/null 2>&1

if [[ -L "target_link.txt" ]]; then
    pass "inheritance worked! symlink created in worktree"
else
    fail "inheritance failed"
fi

popd >/dev/null
# Cleanup worktree
git worktree remove ../feat-1 --force >/dev/null 2>&1

# -----------------------------------------------------------------------------
# Test: global commands (add)
# -----------------------------------------------------------------------------
log "TEST" "Testing 'add' (global) command..."

# Create dummy global source file
echo "GLOBAL_SECRET" > global_source.txt

# Global Add
"$GV_BIN" add global_source.txt
if [[ -f "$TEST_DIR/.git-volume/global_source.txt" ]]; then
    pass "add command copied file to global dir"
else
    fail "add command failed to copy file"
fi

# -----------------------------------------------------------------------------
# Test: global list
# -----------------------------------------------------------------------------
log "TEST" "Testing 'global list' command..."

# Setup more global files for tree test
mkdir -p "$TEST_DIR/.git-volume/secrets/prod"
echo "API_KEY" > "$TEST_DIR/.git-volume/secrets/api.key"
echo "DB_PASS" > "$TEST_DIR/.git-volume/secrets/prod/db.key"

# Run global list
OUTPUT="$("$GV_BIN" global list)"

# Verify output contains expected files
if grep -q "secrets" <<< "$OUTPUT" && \
   grep -q "api.key" <<< "$OUTPUT" && \
   grep -q "db.key" <<< "$OUTPUT" && \
   grep -q "global_source.txt" <<< "$OUTPUT"; then
    pass "global list shows all files"
else
    fail "global list missing files"
    echo "Output: $OUTPUT"
fi

# Verify tree structure
if grep -q "├──" <<< "$OUTPUT" || grep -q "└──" <<< "$OUTPUT"; then
    pass "global list uses tree-style output"
else
    fail "global list missing tree connectors"
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
if [[ $FAILED -eq 0 ]]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "💥 Some tests failed."
    exit 1
fi
