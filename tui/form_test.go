package tui

import (
	"strings"
	"testing"
)

func TestToConnectionValidation(t *testing.T) {
	set := func(f *formModel, field int, val string) { f.inputs[field].SetValue(val) }

	t.Run("name required", func(t *testing.T) {
		f := newFormModel(nil, "")
		set(&f, fieldHost, "example.com")
		if _, _, err := f.toConnection(); err == nil {
			t.Error("expected error for missing name")
		}
	})

	t.Run("host or command required", func(t *testing.T) {
		f := newFormModel(nil, "")
		set(&f, fieldName, "x")
		if _, _, err := f.toConnection(); err == nil {
			t.Error("expected error for missing host and command")
		}
	})

	t.Run("invalid port rejected", func(t *testing.T) {
		for _, bad := range []string{"abc", "0", "70000", "-1"} {
			f := newFormModel(nil, "")
			set(&f, fieldName, "x")
			set(&f, fieldHost, "example.com")
			set(&f, fieldPort, bad)
			if _, _, err := f.toConnection(); err == nil {
				t.Errorf("expected error for port %q", bad)
			}
		}
	})

	t.Run("valid connection with defaults and tag parsing", func(t *testing.T) {
		f := newFormModel(nil, "")
		set(&f, fieldName, "  web-1  ")
		set(&f, fieldHost, "example.com")
		set(&f, fieldTags, " prod, web ,, ")
		conn, group, err := f.toConnection()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if group != "Default" {
			t.Errorf("group = %q, want Default fallback", group)
		}
		if conn.Name != "web-1" {
			t.Errorf("name = %q, want trimmed", conn.Name)
		}
		if strings.Join(conn.Tags, "|") != "prod|web" {
			t.Errorf("tags = %v, want [prod web]", conn.Tags)
		}
	})
}

func TestCommitPortForward(t *testing.T) {
	setPF := func(f *formModel, typ, lp, rh, rp string) {
		f.pfInputs[pfFieldType].SetValue(typ)
		f.pfInputs[pfFieldLocalPort].SetValue(lp)
		f.pfInputs[pfFieldRemoteHost].SetValue(rh)
		f.pfInputs[pfFieldRemotePort].SetValue(rp)
	}

	t.Run("local forward, lowercase type accepted", func(t *testing.T) {
		f := newFormModel(nil, "")
		setPF(&f, "l", "8080", "localhost", "80")
		if err := f.commitPortForward(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pf := f.portForwards[0]
		if pf.Type != "L" || pf.LocalPort != 8080 || pf.RemoteHost != "localhost" || pf.RemotePort != 80 {
			t.Errorf("unexpected forward: %+v", pf)
		}
	})

	t.Run("dynamic forward needs no remote", func(t *testing.T) {
		f := newFormModel(nil, "")
		setPF(&f, "D", "1080", "", "")
		if err := f.commitPortForward(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		f := newFormModel(nil, "")
		setPF(&f, "X", "8080", "localhost", "80")
		if err := f.commitPortForward(); err == nil {
			t.Error("expected error for type X")
		}
	})

	t.Run("missing remote port rejected for local", func(t *testing.T) {
		f := newFormModel(nil, "")
		setPF(&f, "L", "8080", "localhost", "")
		if err := f.commitPortForward(); err == nil {
			t.Error("expected error for missing remote port")
		}
	})
}
