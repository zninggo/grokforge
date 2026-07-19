package main

import (
	"fmt"
	"os"

	"github.com/zninggo/grokforge/internal/buildinfo"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("grokforge %s\n", buildinfo.Version)
		return
	}
	// Phase 0: skeleton only. Full server arrives in later phases.
	fmt.Printf("grokforge %s — skeleton OK (HTTP server not started yet)\n", buildinfo.Version)
	fmt.Println("default listen will be :17890")
}
