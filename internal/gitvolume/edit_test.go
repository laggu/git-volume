package gitvolume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitVolume_GlobalEdit(t *testing.T) {
	// Setup temporary directory for global storage
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global")
	err := os.MkdirAll(globalDir, 0755)
	require.NoError(t, err)

	// Create a dummy GitVolume instance
	gv := &GitVolume{
		ctx: &Context{
			GlobalDir: globalDir,
		},
		verbosity: VerbosityQuiet,
	}

	t.Run("Edit existing file", func(t *testing.T) {
		// Create a file to edit
		targetFile := filepath.Join(globalDir, "config.txt")
		err := os.WriteFile(targetFile, []byte("initial content"), 0644)
		require.NoError(t, err)

		// Mock EDITOR to a script that modifies the file
		// We use printf to avoid portability issues with echo -n
		originalEditor := os.Getenv("EDITOR")
		defer os.Setenv("EDITOR", originalEditor)
		os.Setenv("EDITOR", "sh -c 'printf \" - edited\" >> \"$1\"' --")

		err = gv.GlobalEdit("config.txt")
		assert.NoError(t, err)

		// Verify file content was updated
		content, err := os.ReadFile(targetFile)
		require.NoError(t, err)
		assert.Equal(t, "initial content - edited", string(content))
	})

	t.Run("Edit non-existent file", func(t *testing.T) {
		err := gv.GlobalEdit("missing.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})

	t.Run("Security: Block path traversal", func(t *testing.T) {
		err := gv.GlobalEdit("../outside.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file path")
	})

	t.Run("Global directory does not exist", func(t *testing.T) {
		// Create a GitVolume with non-existent global dir
		gvMissing := &GitVolume{
			ctx: &Context{
				GlobalDir: filepath.Join(tmpDir, "missing-global"),
			},
			verbosity: VerbosityQuiet,
		}
		err := gvMissing.GlobalEdit("anything.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "global directory does not exist")
	})
}
