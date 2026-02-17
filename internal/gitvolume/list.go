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

type globalListState struct {
	globalDir string
	root      *treeNode
}

// GlobalList lists all files in the global directory as a tree.
// Step 1: builds the tree from the file system
// Step 2: prints the tree to stdout
func (g *GitVolume) GlobalList() error {
	state, err := g.beforeAllGlobalList()
	if err != nil {
		return err
	}

	if len(state.root.children) == 0 {
		if !g.quiet {
			fmt.Println("Global storage is empty.")
		}
		return nil
	}

	fmt.Println(state.globalDir)

	for i, child := range state.root.children {
		g.globalList(child, "", i == len(state.root.children)-1)
	}

	return nil
}

func (g *GitVolume) beforeAllGlobalList() (globalListState, error) {
	globalDir := g.ctx.GlobalDir
	root, err := g.buildGlobalTree(globalDir)
	if err != nil {
		return globalListState{}, err
	}
	return globalListState{globalDir: globalDir, root: root}, nil
}

func (g *GitVolume) globalList(node *treeNode, prefix string, isLast bool) {
	printNode(node, prefix, isLast)
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

	// Optimization: Use a map to track nodes at each level for O(1) lookup
	// The tree building logic doesn't strictly need sorted input, so we skip initial sort.

	// Helper map to quickly find nodes by their full path
	nodeMap := make(map[string]*treeNode)
	nodeMap["."] = root

	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		current := root
		currentPath := "."

		for i, part := range parts {
			isLeaf := i == len(parts)-1
			currentPath = currentPath + "/" + part

			child, exists := nodeMap[currentPath]
			if !exists {
				child = &treeNode{name: part, isDir: !isLeaf}
				current.children = append(current.children, child)
				nodeMap[currentPath] = child
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
