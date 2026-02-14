package gitvolume

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// treeNode represents a node in the directory tree
type treeNode struct {
	name     string
	children []*treeNode
	isDir    bool
}

// GlobalList lists all files in the global directory as a tree.
// Step 1: builds the tree from the file system
// Step 2: prints the tree to stdout
func (g *GitVolume) GlobalList() error {
	globalDir := g.ctx.GlobalDir

	// Step 1: build tree
	root, err := g.buildGlobalTree(globalDir)
	if err != nil {
		return err
	}

	if len(root.children) == 0 {
		if !g.quiet {
			fmt.Println("Global storage is empty.")
		}
		return nil
	}

	// Step 2: print tree
	fmt.Println(globalDir)
	for i, child := range root.children {
		printNode(child, "", i == len(root.children)-1)
	}

	return nil
}

// buildGlobalTree walks the global directory and returns a tree structure
func (g *GitVolume) buildGlobalTree(globalDir string) (*treeNode, error) {
	root := &treeNode{name: globalDir, isDir: true}

	if _, err := os.Stat(globalDir); os.IsNotExist(err) {
		return root, nil
	}

	var files []string
	err := filepath.WalkDir(globalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(globalDir, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		current := root
		for i, part := range parts {
			isLeaf := i == len(parts)-1
			var child *treeNode
			for _, c := range current.children {
				if c.name == part {
					child = c
					break
				}
			}
			if child == nil {
				child = &treeNode{name: part, isDir: !isLeaf}
				current.children = append(current.children, child)
			}
			current = child
		}
	}

	sortTree(root)
	return root, nil
}

// sortTree recursively sorts children: directories first, then files, alphabetically
func sortTree(node *treeNode) {
	sort.Slice(node.children, func(i, j int) bool {
		if node.children[i].isDir != node.children[j].isDir {
			return node.children[i].isDir
		}
		return node.children[i].name < node.children[j].name
	})
	for _, child := range node.children {
		sortTree(child)
	}
}

// printNode recursively prints a tree node with connectors
func printNode(node *treeNode, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	fmt.Println(prefix + connector + node.name)

	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}
	for i, child := range node.children {
		printNode(child, childPrefix, i == len(node.children)-1)
	}
}
