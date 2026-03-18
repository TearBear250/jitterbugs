// qaqc is a cross-platform CLI tool that scans git repositories for common
// quality and security issues and emits structured reports.
//
// Usage:
//
//	qaqc scan [flags]
//	qaqc audit-github [flags]
//
// Run "qaqc <command> --help" for flags.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/backbiten/jitterbugs/internal/checks"
	"github.com/backbiten/jitterbugs/internal/core"
	"github.com/backbiten/jitterbugs/internal/report"
)

const usage = `qaqc – QAQC scanner for git repositories

Usage:
  qaqc <command> [flags]

Commands:
  scan           Scan a local repository and print a JSON report to stdout
  audit-github   Audit all repositories for a GitHub organisation or user

Run "qaqc <command> --help" for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "audit-github":
		runAuditGitHub(os.Args[2:])
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
	htmlOut := fs.String("html", "", "Optional path to write an HTML report file")
	_ = fs.Parse(args)

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
