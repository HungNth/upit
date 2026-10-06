package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HungNth/upit/internal/install"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: upit-install <command> [options]\n")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	ctx := context.Background()
	switch cmd {
	case "install":
		runInstall(ctx, args)
	case "uninstall":
		runUninstall(ctx, args)
	case "check-processes":
		runCheckProcesses(ctx, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runInstall(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	installDir := fs.String("install-dir", "", "Target installation directory")
	stagedPayload := fs.String("staged-payload", "", "Staged payload directory")
	stagedLauncher := fs.String("staged-launcher", "", "Path to staged launcher executable")
	stagedUninstaller := fs.String("staged-uninstaller", "", "Path to staged uninstaller executable")
	stagedShortcut := fs.String("staged-shortcut", "", "Path to staged shortcut file")
	version := fs.String("version", "", "Product version (X.Y.Z)")
	silent := fs.Bool("silent", false, "Run non-interactively")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	if *installDir == "" || *stagedPayload == "" {
		fmt.Fprintf(os.Stderr, "missing required flags: --install-dir and --staged-payload\n")
		os.Exit(1)
	}

	opts := install.InstallOptions{
		InstallDir:        filepath.Clean(*installDir),
		StagedPayload:     filepath.Clean(*stagedPayload),
		StagedLauncher:    *stagedLauncher,
		StagedUninstaller: *stagedUninstaller,
		StagedShortcut:    *stagedShortcut,
		Version:           *version,
		Silent:            *silent,
	}
	res, err := install.RunInstall(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "install error: %v\n", err)
		os.Exit(1)
	}

	if res.DeferredCleanup {
		os.Exit(10)
	}
	os.Exit(0)
}

func runUninstall(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	installDir := fs.String("install-dir", "", "Target installation directory")
	silent := fs.Bool("silent", false, "Run non-interactively")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	if *installDir == "" {
		fmt.Fprintf(os.Stderr, "missing required flag: --install-dir\n")
		os.Exit(1)
	}

	opts := install.UninstallOptions{
		InstallDir: filepath.Clean(*installDir),
		Silent:     *silent,
	}

	err := install.RunUninstall(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "uninstall error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runCheckProcesses(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("check-processes", flag.ContinueOnError)
	installDir := fs.String("install-dir", "", "Target installation directory")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	if *installDir == "" {
		fmt.Fprintf(os.Stderr, "missing required flag: --install-dir\n")
		os.Exit(1)
	}

	running, err := install.CheckRunningProcesses(filepath.Clean(*installDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "check processes error: %v\n", err)
		os.Exit(1)
	}

	if len(running) > 0 {
		for _, p := range running {
			fmt.Printf("Process %d: %s\n", p.PID, p.ImagePath)
		}
		os.Exit(2)
	}
	os.Exit(0)
}
