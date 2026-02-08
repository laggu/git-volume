package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyPathWithinBase(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "git-volume-path-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	base := filepath.Join(tmpDir, "workspace")
	require.NoError(t, os.MkdirAll(base, 0755))

	// Create a subdirectory
	subDir := filepath.Join(base, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	externalDir := filepath.Join(tmpDir, "external")
	require.NoError(t, os.MkdirAll(externalDir, 0755))

	tests := []struct {
		name    string
		setup   func() string // returns path to test
		wantErr bool
	}{
		{
			name: "valid path within base",
			setup: func() string {
				return filepath.Join(base, "file.txt")
			},
			wantErr: false,
		},
		{
			name: "valid nested path within base",
			setup: func() string {
				return filepath.Join(base, "subdir", "nested", "file.txt")
			},
			wantErr: false,
		},
		{
			name: "valid path in existing subdir",
			setup: func() string {
				return filepath.Join(subDir, "file.txt")
			},
			wantErr: false,
		},
		{
			name: "symlink within base pointing inside base",
			setup: func() string {
				linkPath := filepath.Join(base, "valid-link")
				require.NoError(t, os.Symlink(subDir, linkPath), "failed to create symlink")
				return filepath.Join(linkPath, "file.txt")
			},
			wantErr: false,
		},
		{
			name: "symlink pointing outside base - direct",
			setup: func() string {
				linkPath := filepath.Join(base, "malicious-link")
				require.NoError(t, os.Symlink(externalDir, linkPath), "failed to create symlink")
				return filepath.Join(linkPath, "file.txt")
			},
			wantErr: true,
		},
		{
			name: "symlink pointing outside base - with non-existent subdirs",
			setup: func() string {
				// This is the specific attack vector from the security review
				linkPath := filepath.Join(base, "attack-link")
				require.NoError(t, os.Symlink(externalDir, linkPath), "failed to create symlink")
				// The "config/settings" part doesn't exist, but the symlink does
				return filepath.Join(linkPath, "config", "settings", "file.txt")
			},
			wantErr: true,
		},
		{
			name: "nested symlink chain escaping base",
			setup: func() string {
				// Create a chain: base/link1 -> base/link2 -> external
				link2 := filepath.Join(base, "link2")
				require.NoError(t, os.Symlink(externalDir, link2), "failed to create symlink")
				link1 := filepath.Join(base, "link1")
				require.NoError(t, os.Symlink(link2, link1), "failed to create symlink")
				return filepath.Join(link1, "file.txt")
			},
			wantErr: true,
		},
		{
			name: "path with .. components",
			setup: func() string {
				return filepath.Join(base, "subdir", "..", "..", "external", "file.txt")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := verifyPathWithinBase(path, base)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyPathWithinBase_SymlinkAttack(t *testing.T) {
	// This test specifically reproduces the attack scenario from the security review
	tmpDir, err := os.MkdirTemp("", "git-volume-symlink-attack-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Setup: workspace (base) and external target
	workspace := filepath.Join(tmpDir, "workspace")
	require.NoError(t, os.MkdirAll(workspace, 0755))

	evilDir := filepath.Join(tmpDir, "evil")
	require.NoError(t, os.MkdirAll(evilDir, 0755))

	// Attack: Create a symlink in workspace pointing to evil directory
	maliciousLink := filepath.Join(workspace, "malicious")
	require.NoError(t, os.Symlink(evilDir, maliciousLink))

	// Target path: workspace/malicious/config/settings/file.txt
	// - "malicious" exists (it's a symlink to evil)
	// - "config/settings" doesn't exist
	// The old vulnerable code would return nil here because EvalSymlinks
	// would fail with os.IsNotExist on "malicious/config"
	targetPath := filepath.Join(maliciousLink, "config", "settings", "file.txt")

	err = verifyPathWithinBase(targetPath, workspace)
	assert.Error(t, err, "verifyPathWithinBase() should have detected symlink escape attack")

	// Verify the error message is informative
	t.Logf("Correctly detected attack with error: %v", err)
}

func TestPathsEqual(t *testing.T) {
	tests := []struct {
		name  string
		path1 string
		path2 string
		want  bool
	}{
		{
			name:  "identical paths",
			path1: "/foo/bar",
			path2: "/foo/bar",
			want:  true,
		},
		{
			name:  "different paths",
			path1: "/foo/bar",
			path2: "/foo/baz",
			want:  false,
		},
		{
			name:  "paths with trailing slash normalization",
			path1: "/foo/bar/",
			path2: "/foo/bar",
			want:  true,
		},
		{
			name:  "paths with . component",
			path1: "/foo/./bar",
			path2: "/foo/bar",
			want:  true,
		},
		{
			name:  "paths with .. component",
			path1: "/foo/baz/../bar",
			path2: "/foo/bar",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pathsEqual(tt.path1, tt.path2))
		})
	}
}
