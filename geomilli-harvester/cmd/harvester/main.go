package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/travayo/auto-complete/internal/daemon"
	"github.com/travayo/auto-complete/internal/envload"
)

func main() {
	envload.Load()

	args := os.Args[1:]
	if len(args) == 0 {
		runServe([]string{})
		return
	}

	cmd := args[0]
	switch cmd {
	case "serve":
		runServe(args[1:])
	case "run":
		runPipeline(args[1:])
	case "help", "-h", "--help":
		printUsage()
	default:
		if strings.HasPrefix(cmd, "-") {
			runServe(args)
		} else {
			fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println("Geomilli Harvester - Streaming OpenStreetMap Ingestion Pipeline & Daemon")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  harvester serve [options]   Start the HTTP/SSE Harvester daemon (default)")
	fmt.Println("  harvester run [options]     Run a one-shot pipeline execution (CLI mode)")
	fmt.Println("  harvester help              Show this help message")
	fmt.Println()
	fmt.Println("Serve options:")
	fmt.Println("  --port <port>               HTTP daemon port (default: $HARVESTER_PORT or $PORT or 8085)")
	fmt.Println("  --meili-host <url>          Meilisearch URL (default: $MEILI_HOST or http://127.0.0.1:7700)")
	fmt.Println("  --meili-key <key>           Meilisearch API key (default: $MEILI_API_KEY)")
	fmt.Println("  --auto-import               Auto-import default countries on startup (default: $AUTO_IMPORT or true)")
	fmt.Println("  --countries <list>          Comma-separated country codes (default: $COUNTRIES or BE)")
	fmt.Println()
	fmt.Println("Run options:")
	fmt.Println("  --stage <stage>             Stage to execute: fetch, build-data, index, run, cleanup (default: run)")
	fmt.Println("  --online-enrich             Enable online Wikidata enrichment (default: $ENRICH_ONLINE or false)")
	fmt.Println("  --concurrency <num>         Number of countries to process in parallel (default: 4)")
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	defaultPort := os.Getenv("HARVESTER_PORT")
	if defaultPort == "" {
		defaultPort = os.Getenv("PORT")
	}
	if defaultPort == "" {
		defaultPort = "8085"
	}

	defaultAutoImport := os.Getenv("AUTO_IMPORT") == "true" || os.Getenv("AUTO_IMPORT") == ""

	var port string
	var meiliHost string
	var meiliKey string
	var autoImport bool
	var countriesEnv string

	fs.StringVar(&port, "port", defaultPort, "Daemon HTTP port")
	fs.StringVar(&meiliHost, "meili-host", getenv("MEILI_HOST", "http://127.0.0.1:7700"), "Meilisearch host URL")
	fs.StringVar(&meiliKey, "meili-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	fs.BoolVar(&autoImport, "auto-import", defaultAutoImport, "Automatically import default countries if index is empty")
	fs.StringVar(&countriesEnv, "countries", getenv("COUNTRIES", "BE"), "Default countries to import")

	_ = fs.Parse(args)

	root, err := os.Getwd()
	if err != nil {
		root = "."
	}

	srv := daemon.NewServer(daemon.ServerConfig{
		Port:         port,
		RootDir:      root,
		MeiliHost:    meiliHost,
		MeiliKey:     meiliKey,
		AutoImport:   autoImport,
		CountriesEnv: countriesEnv,
	})

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "harvester daemon error: %v\n", err)
		os.Exit(1)
	}
}

func runPipeline(args []string) {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}

	bin := filepath.Join("/usr/local/bin", "pipeline")
	var cmd *exec.Cmd
	if fileExists(bin) {
		cmd = exec.Command(bin, args...)
	} else {
		cmdArgs := append([]string{"run", "./cmd/pipeline"}, args...)
		cmd = exec.Command("go", cmdArgs...)
	}

	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
