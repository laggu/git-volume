/*
Copyright © 2026 laggu
*/
package main

import (
	"os"

	"github.com/laggu/git-volume/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
