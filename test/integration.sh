#!/bin/bash

# Configuration
TEST_DIR=$(mktemp -d)
GV_BIN="$PWD/git-volume"
FAILED=0

# Compile git-volume if not present
if [ ! -f "$GV_BIN" ]; then
    echo "🔨 Building git-volume..."
    go build -o git-volume main.go
    if [ $? -ne 0 ]; then
        echo "❌ Build failed"
        exit 1
    fi
     # Use absolute path for GV_BIN
    GV_BIN="$(pwd)/git-volume"
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
cd "$TEST_DIR" || exit 1
mkdir project
cd project || exit 1

# Initialize git repository (required for git-volume)
git init -q

"$GV_BIN" init -q
if [ -d "$TEST_DIR/.git-volume" ] && [ -f "git-volume.yaml" ]; then
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

# Run Sync
"$GV_BIN" sync

# Verify Link
if [ -L "target_link.txt" ]; then
    CONTENT=$(cat target_link.txt)
    if [ "$CONTENT" == "SECRET_DATA" ]; then
        pass "sync created symlink correctly"
    else
        fail "symlink content mismatch"
    fi
else
    fail "sync failed to create symlink"
fi

# Verify Copy
if [ -f "target_copy.txt" ] && [ ! -L "target_copy.txt" ]; then
    CONTENT=$(cat target_copy.txt)
    if [ "$CONTENT" == "SECRET_DATA" ]; then
        pass "sync created copy correctly"
    else
        fail "copy content mismatch"
    fi
else
    fail "sync failed to create copy"
fi

# Run Status
OUTPUT=$("$GV_BIN" status)
if echo "$OUTPUT" | grep -q "target_link.txt" && \
   echo "$OUTPUT" | grep -q "target_copy.txt" && \
   echo "$OUTPUT" | grep -q "OK"; then
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

if [ ! -e "target_link.txt" ]; then
    pass "unsync removed symlink"
else
    fail "unsync failed to remove symlink"
fi

if [ ! -e "target_copy.txt" ]; then
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

if [ -f "target_copy.txt" ]; then
    CONTENT=$(cat target_copy.txt)
    if [ "$CONTENT" == "MODIFIED_DATA" ]; then
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
# Test: global commands (add)
# -----------------------------------------------------------------------------
log "TEST" "Testing 'add' (global) command..."

# Create dummy global source file
echo "GLOBAL_SECRET" > global_source.txt

# Global Add
"$GV_BIN" add global_source.txt
if [ -f "$TEST_DIR/.git-volume/global_source.txt" ]; then
    pass "add command copied file to global dir"
else
    fail "add command failed to copy file"
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
if [ $FAILED -eq 0 ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "💥 Some tests failed."
    exit 1
fi
