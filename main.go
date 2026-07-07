package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavtamm/cssh/config"
	"github.com/taavtamm/cssh/tui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

const usage = `cssh — terminal UI for managing and connecting to SSH hosts

Usage:
  cssh              launch the TUI
  cssh -h, --help   show this help
  cssh -v, --version

Connections are stored in ~/.cssh/config.json and edited through the TUI.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Print(usage)
			return
		case "-v", "--version", "version":
			fmt.Println("cssh " + resolveVersion())
			return
		default:
			fmt.Fprintf(os.Stderr, "cssh: unknown argument %q\n\n%s", os.Args[1], usage)
			os.Exit(2)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cssh: failed to load config: %v\n", err)
		os.Exit(1)
	}

	m := tui.New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cssh: %v\n", err)
		os.Exit(1)
	}

	finalModel, ok := final.(tui.Model)
	if !ok || finalModel.ConnectTo == nil {
		return
	}

	bin, args := finalModel.ConnectTo.BuildArgs()
	binPath, err := exec.LookPath(bin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cssh: %s not found: %v\n", bin, err)
		os.Exit(1)
	}
	if err := syscall.Exec(binPath, append([]string{bin}, args...), os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "cssh: exec failed: %v\n", err)
		os.Exit(1)
	}
}

// resolveVersion prefers the ldflags-injected version, falling back to module
// build info so `go install ...@vX.Y.Z` binaries report their version too.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}
