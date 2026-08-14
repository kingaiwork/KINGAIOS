package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
)

func main() {
	manifestPath := flag.String("manifest", "", "path to a KINGAI OS Device Pack manifest")
	flag.Parse()
	if *manifestPath == "" {
		fmt.Fprintln(os.Stderr, "--manifest is required")
		os.Exit(2)
	}
	manifest, err := devicepack.Load(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("validated Device Pack %s schema=%d arch=%s capabilities=%d\n", manifest.ID, manifest.Schema, manifest.Arch, len(manifest.Capabilities))
}
