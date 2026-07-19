package main

import (
	"fmt"
	"os"

	"github.com/zninggo/grokforge/internal/buildinfo"
)

func main() {
	fmt.Printf("grokforge-migrate %s — migrations land in Phase 1\n", buildinfo.Version)
	if len(os.Args) > 1 && os.Args[1] == "version" {
		return
	}
	os.Exit(0)
}
