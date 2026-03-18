// qaqc is a cross-platform CLI tool that scans a local git repository for
// common quality and security issues and emits structured reports.
//
// Usage:
//
//	qaqc scan [flags]
//
// Flags:
//
//	--path  <dir>   Path to the git repository to scan (default: current directory)
//	--url   <url>   Remote git URL to clone and scan (e.g. https://github.com/owner/repo)
//	--html  <file>  Write an HTML report to this file in addition to JSON stdout
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/backbiten/jitterbugs/internal/checks"
	"github.com/backbiten/jitterbugs/internal/core"
	"github.com/backbiten/jitterbugs/internal/remote"
	"github.com/backbiten/jitterbugs/internal/report"
)

const usage = `qaqc – QAQC scanner for local git repositories

Usage:
  qaqc <command> [flags]

Commands:
  scan    Scan a repository and print a JSON report to stdout

Run "qaqc scan --help" for scan-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "--help", "-h", "help":
		fmt.Fprint(os.Stdout, usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	repoPath := fs.String("path", ".", "Path to the git repository to scan")
	repoURL := fs.String("url", "", "Remote git URL to clone and scan (e.g. https://github.com/owner/repo)")
	htmlOut := fs.String("html", "", "Optional path to write an HTML report file")
	_ = fs.Parse(args)

	if *repoURL != "" {
		if *repoPath != "." {
			fmt.Fprintln(os.Stderr, "error: --url and --path are mutually exclusive; provide one or the other")
			os.Exit(1)
		}
		dir, cleanup, err := remote.Clone(*repoURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error cloning repository: %v\n", err)
			os.Exit(3)
		}
		defer cleanup()
		*repoPath = dir
	}

	cfg := core.LoadConfig(*repoPath)
	runner := core.NewRunner(*repoPath, cfg)
	runner.AddCheck(checks.NewRequiredFilesCheck(cfg))
	runner.AddCheck(checks.NewCIDetectCheck())
	runner.AddCheck(checks.NewSecretsCheck())

	rpt := runner.Run()

	// Always emit JSON to stdout.
	jsonData, err := report.RenderJSON(rpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering JSON report: %v\n", err)
		os.Exit(3)
	}
	fmt.Println(string(jsonData))

	// Optionally write HTML.
	if *htmlOut != "" {
		if err := report.WriteHTML(rpt, *htmlOut); err != nil {
			fmt.Fprintf(os.Stderr, "error writing HTML report: %v\n", err)
		}
	}

	os.Exit(rpt.ExitCode())
}
