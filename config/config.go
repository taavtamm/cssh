package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type PortForward struct {
	Type       string `json:"type"` // "local" (-L), "remote" (-R), "dynamic" (-D)
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"` // empty for dynamic
	RemotePort int    `json:"remote_port"` // empty for dynamic
}

type Connection struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Tags         []string      `json:"tags,omitempty"`
	Command      string        `json:"command,omitempty"`
	Host         string        `json:"host,omitempty"`
	User         string        `json:"user,omitempty"`
	Port         int           `json:"port,omitempty"`
	IdentityFile string        `json:"identity_file,omitempty"`
	ExtraArgs    string        `json:"extra_args,omitempty"`
	PortForwards []PortForward `json:"port_forwards,omitempty"`
}

type Group struct {
	Name        string       `json:"name"`
	Connections []Connection `json:"connections"`
}

type Config struct {
	Groups              []Group `json:"groups"`
	DefaultIdentityFile string  `json:"default_identity_file,omitempty"`
	ThemeName           string  `json:"theme,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cssh", "config.json")
}

func Load() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	// Write to a temp file and rename so a crash mid-write can't corrupt the config.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// BuildArgs returns the binary name and argument list for exec without shell.
// For custom commands, it delegates to "sh -c <command>" intentionally.
func (c *Connection) BuildArgs() (string, []string) {
	if c.Command != "" {
		return "sh", []string{"-c", c.Command}
	}

	var args []string

	if c.Port > 0 && c.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", c.Port))
	}

	if c.IdentityFile != "" {
		args = append(args, "-i", c.IdentityFile)
	}

	for _, pf := range c.PortForwards {
		switch strings.ToLower(pf.Type) {
		case "local", "l":
			args = append(args, "-L", fmt.Sprintf("%d:%s:%d", pf.LocalPort, pf.RemoteHost, pf.RemotePort))
		case "remote", "r":
			args = append(args, "-R", fmt.Sprintf("%d:%s:%d", pf.LocalPort, pf.RemoteHost, pf.RemotePort))
		case "dynamic", "d":
			args = append(args, "-D", fmt.Sprintf("%d", pf.LocalPort))
		}
	}

	if c.ExtraArgs != "" {
		args = append(args, SplitArgs(c.ExtraArgs)...)
	}

	if c.User != "" {
		args = append(args, c.User+"@"+c.Host)
	} else {
		args = append(args, c.Host)
	}

	return "ssh", args
}

// BuildCommand returns the full command as a single string for display/clipboard purposes.
// Arguments are shell-quoted so the result can be pasted into a shell verbatim.
func (c *Connection) BuildCommand() string {
	if c.Command != "" {
		return c.Command
	}

	bin, args := c.BuildArgs()
	parts := []string{bin}
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// SplitArgs splits s on whitespace, honoring single and double quotes, so
// values like -o ProxyCommand="ssh -W %h:%p jump" survive as one argument.
// Backslash escapes are not interpreted.
func SplitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	inToken := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case unicode.IsSpace(r):
			if inToken {
				args = append(args, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if inToken {
		args = append(args, cur.String())
	}
	return args
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`&|;<>(){}[]*?#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ListSSHKeys returns the paths of private key files found in ~/.ssh/.
// It skips public keys, known_hosts, config, and other non-key files.
func ListSSHKeys() []string {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}
	skip := map[string]bool{
		"known_hosts":     true,
		"known_hosts.old": true,
		"authorized_keys": true,
		"config":          true,
		"environment":     true,
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if skip[name] {
			continue
		}
		if strings.HasSuffix(name, ".pub") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		keys = append(keys, filepath.Join(sshDir, name))
	}
	return keys
}

func (pf PortForward) Badge() string {
	switch strings.ToLower(pf.Type) {
	case "local", "l":
		return fmt.Sprintf("L:%d", pf.LocalPort)
	case "remote", "r":
		return fmt.Sprintf("R:%d", pf.LocalPort)
	case "dynamic", "d":
		return fmt.Sprintf("D:%d", pf.LocalPort)
	}
	return ""
}
