package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavtamm/cssh/config"
)

func testConfig() *config.Config {
	return &config.Config{Groups: []config.Group{
		{Name: "Prod", Connections: []config.Connection{
			{Name: "jenkins", Host: "ci.example.com", Tags: []string{"ci"}},
			{Name: "web", Host: "web.example.com"},
		}},
		{Name: "Dev", Connections: []config.Connection{
			{Name: "backup", Host: "backup.example.com"},
		}},
	}}
}

func press(t *testing.T, m Model, keys ...tea.KeyMsg) Model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(k)
		var ok bool
		m, ok = next.(Model)
		if !ok {
			t.Fatalf("Update returned %T, want Model", next)
		}
	}
	return m
}

func runes(s string) []tea.KeyMsg {
	var msgs []tea.KeyMsg
	for _, r := range s {
		msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return msgs
}

// Regression: j/k/every printable rune must be typeable in the search query,
// not swallowed by list navigation.
func TestSearchAcceptsJAndK(t *testing.T) {
	m := New(testConfig())
	m = press(t, m, runes("/")...)
	if !m.searchActive {
		t.Fatal("expected search to be active after /")
	}

	m = press(t, m, runes("jenkins")...)
	if m.searchQuery != "jenkins" {
		t.Fatalf("searchQuery = %q, want %q", m.searchQuery, "jenkins")
	}

	fi := m.filteredItems()
	// Expect exactly the Prod group header plus the jenkins connection.
	if len(fi) != 2 || !fi[0].isGroup || fi[1].conn == nil || fi[1].conn.Name != "jenkins" {
		t.Errorf("filtered items = %+v, want Prod header + jenkins", fi)
	}
}

func TestSearchArrowsStillNavigate(t *testing.T) {
	m := New(testConfig())
	m = press(t, m, runes("/")...)
	before := m.cursor
	m = press(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor == before {
		t.Error("down arrow should move the cursor while searching")
	}
	if m.searchQuery != "" {
		t.Errorf("arrow key must not modify the query, got %q", m.searchQuery)
	}
}

func TestSearchEscClearsFilter(t *testing.T) {
	m := New(testConfig())
	m = press(t, m, runes("/web")...)
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchActive || m.searchQuery != "" {
		t.Errorf("esc should deactivate and clear search, got active=%v query=%q", m.searchActive, m.searchQuery)
	}
}

func TestMatchesSearch(t *testing.T) {
	conn := &config.Connection{
		Name: "web-1", Host: "10.0.0.1", User: "deploy",
		Description: "primary frontend", Tags: []string{"prod", "web"},
	}
	for _, q := range []string{"web-1", "10.0.0", "deploy", "frontend", "prod", "mygroup"} {
		if !matchesSearch(conn, "MyGroup", q) {
			t.Errorf("expected query %q to match", q)
		}
	}
	if matchesSearch(conn, "MyGroup", "database") {
		t.Error("expected query 'database' not to match")
	}
}

func TestMoveCursorSkipsGroupHeadersAndWraps(t *testing.T) {
	m := New(testConfig())
	// Cursor starts on the first connection (index 1; index 0 is a group header).
	if m.cursor != 1 {
		t.Fatalf("initial cursor = %d, want 1", m.cursor)
	}
	m.moveCursor(-1) // wrap to the last connection
	fi := m.filteredItems()
	if fi[m.cursor].isGroup || fi[m.cursor].conn.Name != "backup" {
		t.Errorf("expected wrap to last connection, got item %+v", fi[m.cursor])
	}
	m.moveCursor(1) // wrap forward again, skipping headers
	if fi[m.cursor].isGroup || fi[m.cursor].conn.Name != "jenkins" {
		t.Errorf("expected wrap to first connection, got item %+v", fi[m.cursor])
	}
}

// Regression: moving a connection to another group must remove it by index,
// not by name, so duplicates don't get the wrong entry deleted.
func TestSaveConnectionMovesCorrectDuplicate(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // sandbox the config write

	cfg := &config.Config{Groups: []config.Group{
		{Name: "A", Connections: []config.Connection{
			{Name: "web", Host: "first"},
			{Name: "web", Host: "second"},
		}},
	}}
	m := New(cfg)
	m.state = stateEdit
	item := listItem{groupIdx: 0, connIdx: 1, groupName: "A", conn: &cfg.Groups[0].Connections[1]}
	m.editingItem = &item

	moved := cfg.Groups[0].Connections[1]
	m.saveConnection(&moved, "B")

	if len(m.cfg.Groups[0].Connections) != 1 || m.cfg.Groups[0].Connections[0].Host != "first" {
		t.Errorf("wrong connection removed from group A: %+v", m.cfg.Groups[0].Connections)
	}
	last := m.cfg.Groups[len(m.cfg.Groups)-1]
	if last.Name != "B" || len(last.Connections) != 1 || last.Connections[0].Host != "second" {
		t.Errorf("moved connection not in group B: %+v", m.cfg.Groups)
	}
}

func TestDeleteSelectedRemovesEmptyGroup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := New(testConfig())
	// Move cursor to the only Dev connection ("backup", last item).
	m.moveCursor(-1)
	m.deleteSelected()

	for _, g := range m.cfg.Groups {
		if g.Name == "Dev" {
			t.Error("empty Dev group should have been removed")
		}
	}
}

func TestStaleStatusClearDoesNotWipeNewerMessage(t *testing.T) {
	m := New(testConfig())
	_ = m.setStatus("first")
	staleGen := m.statusGen
	_ = m.setStatus("second")

	next, _ := m.Update(clearStatusMsg{gen: staleGen})
	m = next.(Model)
	if m.statusMsg != "second" {
		t.Errorf("stale clear wiped newer message, statusMsg = %q", m.statusMsg)
	}

	next, _ = m.Update(clearStatusMsg{gen: m.statusGen})
	m = next.(Model)
	if m.statusMsg != "" {
		t.Errorf("current-gen clear should blank the message, got %q", m.statusMsg)
	}
}
