package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		conn     Connection
		wantBin  string
		wantArgs []string
	}{
		{
			name:     "host only",
			conn:     Connection{Host: "example.com"},
			wantBin:  "ssh",
			wantArgs: []string{"example.com"},
		},
		{
			name:     "user at host",
			conn:     Connection{Host: "example.com", User: "ubuntu"},
			wantBin:  "ssh",
			wantArgs: []string{"ubuntu@example.com"},
		},
		{
			name:     "non-default port",
			conn:     Connection{Host: "example.com", Port: 2222},
			wantBin:  "ssh",
			wantArgs: []string{"-p", "2222", "example.com"},
		},
		{
			name:     "default port omitted",
			conn:     Connection{Host: "example.com", Port: 22},
			wantBin:  "ssh",
			wantArgs: []string{"example.com"},
		},
		{
			name:     "identity file",
			conn:     Connection{Host: "example.com", IdentityFile: "~/.ssh/id_ed25519"},
			wantBin:  "ssh",
			wantArgs: []string{"-i", "~/.ssh/id_ed25519", "example.com"},
		},
		{
			name: "port forwards short and long type spellings",
			conn: Connection{
				Host: "example.com",
				PortForwards: []PortForward{
					{Type: "L", LocalPort: 8080, RemoteHost: "localhost", RemotePort: 80},
					{Type: "remote", LocalPort: 9090, RemoteHost: "127.0.0.1", RemotePort: 9091},
					{Type: "dynamic", LocalPort: 1080},
				},
			},
			wantBin:  "ssh",
			wantArgs: []string{"-L", "8080:localhost:80", "-R", "9090:127.0.0.1:9091", "-D", "1080", "example.com"},
		},
		{
			name:     "extra args with quoted option",
			conn:     Connection{Host: "example.com", ExtraArgs: `-o ProxyCommand="ssh -W %h:%p jump"`},
			wantBin:  "ssh",
			wantArgs: []string{"-o", "ProxyCommand=ssh -W %h:%p jump", "example.com"},
		},
		{
			name:     "custom command delegates to sh",
			conn:     Connection{Command: "kubectl exec -it pod -- bash"},
			wantBin:  "sh",
			wantArgs: []string{"-c", "kubectl exec -it pod -- bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin, args := tt.conn.BuildArgs()
			if bin != tt.wantBin {
				t.Errorf("bin = %q, want %q", bin, tt.wantBin)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %q, want %q", args, tt.wantArgs)
			}
		})
	}
}

func TestBuildCommandQuotesSpecialArgs(t *testing.T) {
	conn := Connection{Host: "example.com", IdentityFile: "/keys/my key"}
	got := conn.BuildCommand()
	want := `ssh -i '/keys/my key' example.com`
	if got != want {
		t.Errorf("BuildCommand() = %q, want %q", got, want)
	}

	plain := Connection{Host: "example.com", User: "root"}
	if got := plain.BuildCommand(); got != "ssh root@example.com" {
		t.Errorf("BuildCommand() = %q, want unquoted plain args", got)
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"-A -X", []string{"-A", "-X"}},
		{`-o ProxyCommand="ssh -W %h:%p jump"`, []string{"-o", "ProxyCommand=ssh -W %h:%p jump"}},
		{`-o 'StrictHostKeyChecking no'`, []string{"-o", "StrictHostKeyChecking no"}},
		{`"unterminated quote`, []string{"unterminated quote"}},
	}
	for _, tt := range tests {
		if got := SplitArgs(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitArgs(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := &Config{
		ThemeName:           "Tokyo Night",
		DefaultIdentityFile: "~/.ssh/id_ed25519",
		Groups: []Group{{
			Name: "Prod",
			Connections: []Connection{{
				Name: "web-1",
				Host: "10.0.0.1",
				User: "deploy",
				Port: 2222,
				Tags: []string{"prod", "web"},
				PortForwards: []PortForward{
					{Type: "L", LocalPort: 5432, RemoteHost: "localhost", RemotePort: 5432},
				},
			}},
		}},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The atomic-write temp file must not be left behind.
	if _, err := os.Stat(configPath() + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind after Save")
	}

	info, err := os.Stat(configPath())
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file mode = %o, want 0600", perm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Errorf("round trip mismatch:\ngot  %+v\nwant %+v", got, cfg)
	}
}

func TestLoadMissingConfigReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Groups) != 0 {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoadCorruptConfigErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".cssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("expected error for corrupt config, got nil")
	}
}
