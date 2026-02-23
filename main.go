package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavitammiste/cssh/config"
	"github.com/taavitammiste/cssh/tui"
)

func main() {
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
