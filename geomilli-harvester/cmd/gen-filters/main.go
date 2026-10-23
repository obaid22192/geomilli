package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/travayo/auto-complete/internal/envload"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

func main() {
	envload.Load()

	var taxonomyPath string
	var layer string

	flag.StringVar(&taxonomyPath, "taxonomy", "config/taxonomy.yaml", "Path to taxonomy YAML file")
	flag.StringVar(&layer, "layer", "", "Filter layer to output: admin or poi (required)")
	flag.Parse()

	if layer == "" {
		fmt.Fprintln(os.Stderr, "error: --layer is required (admin or poi)")
		os.Exit(1)
	}

	tax, err := taxonomy.LoadFromFile(taxonomyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load taxonomy: %v\n", err)
		os.Exit(1)
	}

	var args []string
	switch layer {
	case "admin":
		args = tax.AdminOsmiumArgs()
	case "poi":
		args = tax.POIOsmiumArgs()
	default:
		fmt.Fprintf(os.Stderr, "error: invalid layer %q (must be admin or poi)\n", layer)
		os.Exit(1)
	}

	fmt.Println(strings.Join(args, " "))
}
