package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavitammiste/cssh/config"
)

type clearStatusMsg struct{}

type state int

const (
	stateList state = iota
	stateAdd
	stateEdit
	stateConfirmDelete
	stateAddPortFwd
	stateKeyPicker
	stateDetail
)

// listItem represents a group header or a connection row.
type listItem struct {
	isGroup   bool
	groupIdx  int
	connIdx   int
	groupName string
	conn      *config.Connection
}

type Model struct {
	cfg           *config.Config
	items         []listItem
	cursor        int
	state         state
	form          formModel
	width         int
	height        int
	ConnectTo     *config.Connection
	availableKeys []string
	keyPickerIdx  int
	searchActive  bool
	searchQuery   string
	editingItem   *listItem
	statusMsg     string // transient message (e.g. "Copied!")
}

func New(cfg *config.Config) Model {
	if cfg.ThemeName != "" {
		SetThemeByName(cfg.ThemeName)
	}
	m := Model{cfg: cfg}
	m.rebuildItems()
	m.resetCursor()
	return m
}

// ── Item management ────────────────────────────────────────────────────────

func (m *Model) rebuildItems() {
	m.items = nil
	for gi, g := range m.cfg.Groups {
		m.items = append(m.items, listItem{isGroup: true, groupIdx: gi, groupName: g.Name})
		for ci := range g.Connections {
			conn := &m.cfg.Groups[gi].Connections[ci]
			m.items = append(m.items, listItem{groupIdx: gi, connIdx: ci, conn: conn, groupName: g.Name})
		}
	}
}

func (m Model) filteredItems() []listItem {
	if m.searchQuery == "" {
		return m.items
	}
	q := strings.ToLower(m.searchQuery)
	matched := map[int]bool{}
	for _, item := range m.items {
		if !item.isGroup && item.conn != nil && matchesSearch(item.conn, item.groupName, q) {
			matched[item.groupIdx] = true
		}
	}
	var out []listItem
	for _, item := range m.items {
		if item.isGroup {
			if matched[item.groupIdx] {
				out = append(out, item)
			}
		} else if item.conn != nil && matchesSearch(item.conn, item.groupName, q) {
			out = append(out, item)
		}
	}
	return out
}

func matchesSearch(conn *config.Connection, group, q string) bool {
	fields := []string{conn.Name, conn.Host, conn.User, conn.Description, group}
	fields = append(fields, conn.Tags...)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// resetCursor moves the cursor to the first selectable item in the filtered list.
func (m *Model) resetCursor() {
	for i, item := range m.filteredItems() {
		if !item.isGroup {
			m.cursor = i
			return
		}
	}
	m.cursor = 0
}

func (m *Model) moveCursor(dir int) {
	fi := m.filteredItems()
	if len(fi) == 0 {
		return
	}
	pos := m.cursor + dir
	for range fi {
		if pos < 0 {
			pos = len(fi) - 1
		} else if pos >= len(fi) {
			pos = 0
		}
		if !fi[pos].isGroup {
			m.cursor = pos
			return
		}
		pos += dir
	}
}

// ── Bubble Tea ─────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(clearStatusMsg); ok {
		m.statusMsg = ""
		return m, nil
	}
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateAdd, stateEdit:
		return m.updateForm(msg)
	case stateConfirmDelete:
		return m.updateConfirm(msg)
	case stateAddPortFwd:
		return m.updatePortFwdForm(msg)
	case stateKeyPicker:
		return m.updateKeyPicker(msg)
	case stateDetail:
		return m.updateDetail(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.searchActive {
			return m.updateSearch(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)

		case "enter", " ":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				if item := fi[m.cursor]; !item.isGroup && item.conn != nil {
					m.ConnectTo = item.conn
					return m, tea.Quit
				}
			}

		case "y":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				if item := fi[m.cursor]; !item.isGroup && item.conn != nil {
					if err := copyToClipboard(item.conn.BuildCommand()); err != nil {
						m.statusMsg = "Copy failed: " + err.Error()
					} else {
						m.statusMsg = "Command copied!"
					}
					return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return clearStatusMsg{}
					})
				}
			}

		case "i":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				if item := fi[m.cursor]; !item.isGroup && item.conn != nil {
					m.state = stateDetail
					m.statusMsg = ""
				}
			}

		case "c":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				if item := fi[m.cursor]; !item.isGroup && item.conn != nil {
					cloned := *item.conn
					cloned.Name = cloned.Name + " (copy)"
					if len(cloned.PortForwards) > 0 {
						pfs := make([]config.PortForward, len(cloned.PortForwards))
						copy(pfs, cloned.PortForwards)
						cloned.PortForwards = pfs
					}
					m.availableKeys = config.ListSSHKeys()
					m.keyPickerIdx = 0
					m.form = newFormModel(&cloned, item.groupName)
					m.form.isEdit = false
					m.state = stateAdd
					m.statusMsg = ""
				}
			}

		case "a":
			m.availableKeys = config.ListSSHKeys()
			m.keyPickerIdx = 0
			m.form = newFormModel(nil, "")
			if m.cfg.DefaultIdentityFile != "" {
				m.form.inputs[fieldIdentity].SetValue(m.cfg.DefaultIdentityFile)
			}
			m.state = stateAdd
			m.statusMsg = ""

		case "e":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				item := fi[m.cursor]
				if !item.isGroup && item.conn != nil {
					m.availableKeys = config.ListSSHKeys()
					m.keyPickerIdx = 0
					m.editingItem = &item
					m.form = newFormModel(item.conn, item.groupName)
					m.state = stateEdit
					m.statusMsg = ""
				}
			}

		case "d":
			fi := m.filteredItems()
			if m.cursor < len(fi) && !fi[m.cursor].isGroup {
				m.state = stateConfirmDelete
				m.statusMsg = ""
			}

		case "/":
			m.searchActive = true
			m.statusMsg = ""

		case "T":
			theme := NextTheme()
			m.cfg.ThemeName = theme.Name
			config.Save(m.cfg)
			m.statusMsg = "Theme: " + theme.Name
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
	}
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""
		m.resetCursor()
	case "enter":
		m.searchActive = false
	case "backspace", "ctrl+h":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.resetCursor()
		}
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	default:
		if s := msg.String(); utf8.RuneCountInString(s) == 1 {
			m.searchQuery += s
			m.resetCursor()
		}
	}
	return m, nil
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "ctrl+s":
			conn, groupName, err := m.form.toConnection()
			if err != nil {
				m.form.errMsg = err.Error()
				return m, nil
			}
			m.saveConnection(conn, groupName)
			m.state = stateList
			return m, nil
		case "ctrl+f":
			m.form.clearPFInputs()
			m.form.editingPF = false
			m.state = stateAddPortFwd
			return m, nil
		case "ctrl+r":
			if len(m.form.portForwards) > 0 {
				m.form.portForwards = m.form.portForwards[:len(m.form.portForwards)-1]
			}
			return m, nil
		case "ctrl+k":
			m.keyPickerIdx = 0
			current := m.form.inputs[fieldIdentity].Value()
			for i, k := range m.availableKeys {
				if k == current {
					m.keyPickerIdx = i
					break
				}
			}
			m.state = stateKeyPicker
			return m, nil
		case "tab", "down":
			m.form.focusField((m.form.focused + 1) % fieldCount)
			return m, nil
		case "shift+tab", "up":
			m.form.focusField((m.form.focused - 1 + fieldCount) % fieldCount)
			return m, nil
		}
	}
	cmd := m.form.updateInputs(msg)
	return m, cmd
}

func (m Model) updatePortFwdForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	back := func() state {
		if m.form.isEdit {
			return stateEdit
		}
		return stateAdd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.form.editingPF = false
			m.form.errMsg = ""
			m.state = back()
			return m, nil
		case "enter":
			if err := m.form.commitPortForward(); err != nil {
				m.form.errMsg = err.Error()
				return m, nil
			}
			m.form.clearPFInputs()
			m.form.errMsg = ""
			m.form.editingPF = false
			m.state = back()
			return m, nil
		case "tab", "down":
			m.form.focusPFField((m.form.pfFocused + 1) % pfFieldCount)
			return m, nil
		case "shift+tab", "up":
			m.form.focusPFField((m.form.pfFocused - 1 + pfFieldCount) % pfFieldCount)
			return m, nil
		}
	}
	m.form.editingPF = true
	return m, m.form.updateInputs(msg)
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			m.deleteSelected()
			m.state = stateList
		case "n", "N", "esc":
			m.state = stateList
		}
	}
	return m, nil
}

func (m Model) updateKeyPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	back := func() state {
		if m.form.isEdit {
			return stateEdit
		}
		return stateAdd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = back()
		case "up", "k":
			if m.keyPickerIdx > 0 {
				m.keyPickerIdx--
			}
		case "down", "j":
			if m.keyPickerIdx < len(m.availableKeys)-1 {
				m.keyPickerIdx++
			}
		case "enter":
			if m.keyPickerIdx < len(m.availableKeys) {
				m.form.inputs[fieldIdentity].SetValue(m.availableKeys[m.keyPickerIdx])
			}
			m.state = back()
		case "ctrl+d":
			if m.keyPickerIdx < len(m.availableKeys) {
				key := m.availableKeys[m.keyPickerIdx]
				m.form.inputs[fieldIdentity].SetValue(key)
				m.cfg.DefaultIdentityFile = key
				config.Save(m.cfg)
			}
			m.state = back()
		}
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.state = stateList
		case "enter":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				if item := fi[m.cursor]; !item.isGroup && item.conn != nil {
					m.ConnectTo = item.conn
					return m, tea.Quit
				}
			}
		case "e":
			fi := m.filteredItems()
			if m.cursor < len(fi) {
				item := fi[m.cursor]
				if !item.isGroup && item.conn != nil {
					m.availableKeys = config.ListSSHKeys()
					m.keyPickerIdx = 0
					m.editingItem = &item
					m.form = newFormModel(item.conn, item.groupName)
					m.state = stateEdit
				}
			}
		}
	}
	return m, nil
}

// ── Data mutations ─────────────────────────────────────────────────────────

func (m *Model) saveConnection(conn *config.Connection, groupName string) {
	if m.state == stateEdit && m.editingItem != nil {
		item := m.editingItem
		if item.groupName != groupName {
			// Remove from old group
			for gi := range m.cfg.Groups {
				if m.cfg.Groups[gi].Name == item.groupName {
					conns := m.cfg.Groups[gi].Connections
					for ci, c := range conns {
						if c.Name == item.conn.Name {
							m.cfg.Groups[gi].Connections = append(conns[:ci], conns[ci+1:]...)
							break
						}
					}
					if len(m.cfg.Groups[gi].Connections) == 0 {
						m.cfg.Groups = append(m.cfg.Groups[:gi], m.cfg.Groups[gi+1:]...)
					}
					break
				}
			}
			m.addToGroup(conn, groupName)
		} else {
			m.cfg.Groups[item.groupIdx].Connections[item.connIdx] = *conn
		}
	} else {
		m.addToGroup(conn, groupName)
	}
	m.editingItem = nil
	m.rebuildItems()
	config.Save(m.cfg)
	// Park cursor on the saved item
	for i, item := range m.filteredItems() {
		if !item.isGroup && item.conn != nil && item.conn.Name == conn.Name && item.groupName == groupName {
			m.cursor = i
			return
		}
	}
}

func (m *Model) addToGroup(conn *config.Connection, groupName string) {
	for gi := range m.cfg.Groups {
		if m.cfg.Groups[gi].Name == groupName {
			m.cfg.Groups[gi].Connections = append(m.cfg.Groups[gi].Connections, *conn)
			return
		}
	}
	m.cfg.Groups = append(m.cfg.Groups, config.Group{
		Name:        groupName,
		Connections: []config.Connection{*conn},
	})
}

func (m *Model) deleteSelected() {
	fi := m.filteredItems()
	if m.cursor >= len(fi) {
		return
	}
	item := fi[m.cursor]
	if item.isGroup {
		return
	}
	gi, ci := item.groupIdx, item.connIdx
	m.cfg.Groups[gi].Connections = append(
		m.cfg.Groups[gi].Connections[:ci],
		m.cfg.Groups[gi].Connections[ci+1:]...,
	)
	if len(m.cfg.Groups[gi].Connections) == 0 {
		m.cfg.Groups = append(m.cfg.Groups[:gi], m.cfg.Groups[gi+1:]...)
	}
	m.rebuildItems()
	config.Save(m.cfg)
	m.resetCursor()
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// ── Views ──────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.state {
	case stateAdd:
		return m.form.viewForm("Add Connection", m.width)
	case stateEdit:
		return m.form.viewForm("Edit Connection", m.width)
	case stateAddPortFwd:
		return m.form.viewPortForwardForm(m.width)
	case stateConfirmDelete:
		return m.viewConfirm()
	case stateKeyPicker:
		return m.viewKeyPicker()
	case stateDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

func (m Model) viewList() string {
	var sb strings.Builder

	// Box dimensions — full terminal width.
	boxW := m.width
	if boxW < 44 {
		boxW = 44
	}
	inner := boxW - 2 // content width between the │ chars

	// boxLine pads styled content to fill inner width then wraps with │ borders.
	boxLine := func(content string) string {
		w := lipgloss.Width(content)
		pad := inner - w
		if pad < 0 {
			pad = 0
		}
		return borderStyle.Render("│") + content + strings.Repeat(" ", pad) + borderStyle.Render("│")
	}

	// ── Top border with title embedded ──
	titleText := " cssh · ssh connection manager "
	titleStyled := titleStyle.Render(titleText)
	borderFill := inner - 1 - lipgloss.Width(titleStyled)
	if borderFill < 0 {
		borderFill = 0
	}
	sb.WriteString(borderStyle.Render("╭─") + titleStyled + borderStyle.Render(strings.Repeat("─", borderFill)+"╮") + "\n")
	sb.WriteString(boxLine("") + "\n")

	// ── Connection list ──
	fi := m.filteredItems()
	if len(fi) == 0 {
		if m.searchQuery != "" {
			sb.WriteString(boxLine(helpStyle.Render("  No matches for \""+m.searchQuery+"\"")) + "\n")
		} else {
			sb.WriteString(boxLine(helpStyle.Render("  No connections — press 'a' to add one.")) + "\n")
		}
	} else {
		for i, item := range fi {
			sb.WriteString(boxLine(m.renderItem(i, item, fi)) + "\n")
		}
	}

	sb.WriteString(boxLine("") + "\n")

	// ── Middle divider ──
	sb.WriteString(borderStyle.Render("├"+strings.Repeat("─", inner)+"┤") + "\n")

	// ── Footer ──
	if m.searchActive {
		sb.WriteString(boxLine(searchActiveStyle.Render("  /"+m.searchQuery+"█")) + "\n")
		sb.WriteString(boxLine(helpStyle.Render("  type to filter  ↑↓ navigate  enter confirm  esc clear")) + "\n")
	} else {
		sb.WriteString(boxLine(helpStyle.Render("  ↑↓/jk move  enter connect  i detail  / search  y copy  c dup  a add  e edit  d del  T theme  q quit")) + "\n")
		if m.searchQuery != "" {
			sb.WriteString(boxLine(helpStyle.Render("  filter: "+m.searchQuery)) + "\n")
		}
		if m.statusMsg != "" {
			sb.WriteString(boxLine(successStyle.Render("  "+m.statusMsg)) + "\n")
		}
		sb.WriteString(boxLine(themeNameStyle.Render("  "+CurrentTheme().Name)) + "\n")
	}

	// ── Bottom border ──
	sb.WriteString(borderStyle.Render("╰"+strings.Repeat("─", inner)+"╯") + "\n")

	return sb.String()
}

func (m Model) viewDetail() string {
	fi := m.filteredItems()
	if m.cursor >= len(fi) {
		return ""
	}
	item := fi[m.cursor]
	if item.isGroup || item.conn == nil {
		return ""
	}
	conn := item.conn

	boxW := m.width
	if boxW < 50 {
		boxW = 50
	}
	inner := boxW - 2

	boxLine := func(content string) string {
		w := lipgloss.Width(content)
		pad := inner - w
		if pad < 0 {
			pad = 0
		}
		return borderStyle.Render("│") + content + strings.Repeat(" ", pad) + borderStyle.Render("│")
	}

	var sb strings.Builder

	// Top border with title
	titleText := " Connection Details "
	titleStyled := titleStyle.Render(titleText)
	borderFill := inner - 1 - lipgloss.Width(titleStyled)
	if borderFill < 0 {
		borderFill = 0
	}
	sb.WriteString(borderStyle.Render("╭─") + titleStyled + borderStyle.Render(strings.Repeat("─", borderFill)+"╮") + "\n")
	sb.WriteString(boxLine("") + "\n")

	// Detail rows
	field := func(label, value string) {
		if value == "" {
			return
		}
		styled := formLabelStyle.Render(fmt.Sprintf("  %-16s", label)) + connNameStyle.Render(value)
		sb.WriteString(boxLine(styled) + "\n")
	}

	field("Name", conn.Name)
	field("Group", item.groupName)

	// Host info
	hostInfo := ""
	if conn.Host != "" {
		if conn.User != "" {
			hostInfo = conn.User + "@" + conn.Host
		} else {
			hostInfo = conn.Host
		}
		if conn.Port > 0 && conn.Port != 22 {
			hostInfo += fmt.Sprintf(" :%d", conn.Port)
		}
	}
	field("Host", hostInfo)
	field("Identity", conn.IdentityFile)

	// Tags
	if len(conn.Tags) > 0 {
		var tagBadges []string
		for _, tag := range conn.Tags {
			tagBadges = append(tagBadges, tagStyle(tag).Render(tag))
		}
		styled := formLabelStyle.Render(fmt.Sprintf("  %-16s", "Tags")) + strings.Join(tagBadges, " ")
		sb.WriteString(boxLine(styled) + "\n")
	}

	field("Extra Args", conn.ExtraArgs)
	field("Command", conn.Command)
	field("Description", conn.Description)

	// Port forwards
	if len(conn.PortForwards) > 0 {
		sb.WriteString(boxLine("") + "\n")
		sb.WriteString(boxLine(formSectionStyle.Render("  Port Forwards")) + "\n")
		for _, pf := range conn.PortForwards {
			var pfStr string
			switch strings.ToUpper(pf.Type) {
			case "D":
				pfStr = fmt.Sprintf("D:%d", pf.LocalPort)
			default:
				pfStr = fmt.Sprintf("%s:%d → %s:%d", strings.ToUpper(pf.Type), pf.LocalPort, pf.RemoteHost, pf.RemotePort)
			}
			sb.WriteString(boxLine(pfItemStyle.Render("  • "+pfStr)) + "\n")
		}
	}

	// SSH Command
	sb.WriteString(boxLine("") + "\n")
	sb.WriteString(boxLine(formSectionStyle.Render("  SSH Command")) + "\n")
	sb.WriteString(boxLine(connHostStyle.Render("  "+conn.BuildCommand())) + "\n")

	sb.WriteString(boxLine("") + "\n")

	// Footer
	sb.WriteString(borderStyle.Render("├"+strings.Repeat("─", inner)+"┤") + "\n")
	sb.WriteString(boxLine(helpStyle.Render("  enter connect  e edit  esc back")) + "\n")
	sb.WriteString(borderStyle.Render("╰"+strings.Repeat("─", inner)+"╯") + "\n")

	return sb.String()
}

func (m Model) renderItem(idx int, item listItem, fi []listItem) string {
	isSelected := idx == m.cursor && !item.isGroup

	if item.isGroup {
		count := len(m.cfg.Groups[item.groupIdx].Connections)
		return "  " + groupHeaderStyle.Render(fmt.Sprintf("○  %s (%d)", item.groupName, count))
	}

	// Tree connector: is this the last connection in its group within the displayed list?
	isLast := true
	for j := idx + 1; j < len(fi); j++ {
		if fi[j].isGroup {
			break
		}
		if fi[j].groupIdx == item.groupIdx {
			isLast = false
			break
		}
	}
	connector := "├  "
	if isLast {
		connector = "└  "
	}

	marker := "  "
	if isSelected {
		marker = markerStyle.Render("▶ ")
	}

	// Port-forward badges
	var pfBadges []string
	for _, pf := range item.conn.PortForwards {
		pfBadges = append(pfBadges, pf.Badge())
	}
	pfStr := ""
	if len(pfBadges) > 0 {
		pfStr = " " + badgeStyle.Render("[→"+strings.Join(pfBadges, " ")+"]")
	}

	// Tag badges
	tagStr := ""
	for _, tag := range item.conn.Tags {
		tagStr += " " + tagStyle(tag).Render(tag)
	}

	// Host info
	hostInfo := ""
	if item.conn.Command != "" {
		hostInfo = item.conn.Command
	} else if item.conn.Host != "" {
		if item.conn.User != "" {
			hostInfo = item.conn.User + "@" + item.conn.Host
		} else {
			hostInfo = item.conn.Host
		}
		if item.conn.Port > 0 && item.conn.Port != 22 {
			hostInfo += fmt.Sprintf(" :%d", item.conn.Port)
		}
	}

	if isSelected {
		prefix := marker + selectedStyle.Render("│  "+connector)
		return prefix + selectedNameStyle.Render(item.conn.Name) +
			selectedHostStyle.Render("  "+hostInfo) + pfStr + tagStr
	}
	prefix := marker + connHostStyle.Render("│  "+connector)
	return prefix + connNameStyle.Render(item.conn.Name) +
		connHostStyle.Render("  "+hostInfo) + pfStr + tagStr
}

func (m Model) viewConfirm() string {
	fi := m.filteredItems()
	if m.cursor >= len(fi) {
		return ""
	}
	item := fi[m.cursor]
	if item.isGroup || item.conn == nil {
		return ""
	}
	return "\n" + confirmStyle.Render(fmt.Sprintf("  Delete '%s'? (y/n)", item.conn.Name)) + "\n"
}

func (m Model) viewKeyPicker() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(formTitleStyle.Render("  Select Identity File") + "\n")

	lineW := min(m.width-4, 50)
	if lineW < 20 {
		lineW = 40
	}
	sb.WriteString(dividerStyle.Render("  "+strings.Repeat("─", lineW)) + "\n\n")

	if len(m.availableKeys) == 0 {
		sb.WriteString(helpStyle.Render("  No keys found in ~/.ssh/") + "\n")
	} else {
		for i, key := range m.availableKeys {
			name := filepath.Base(key)
			defaultTag := ""
			if key == m.cfg.DefaultIdentityFile {
				defaultTag = " " + successStyle.Render("(default)")
			}
			if i == m.keyPickerIdx {
				sb.WriteString("  " + markerStyle.Render("▶ ") + selectedNameStyle.Render(name) + defaultTag + "\n")
			} else {
				sb.WriteString("     " + connNameStyle.Render(name) + defaultTag + "\n")
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(dividerStyle.Render("  "+strings.Repeat("─", lineW)) + "\n")
	sb.WriteString(helpStyle.Render("  ↑↓/jk move  enter select  ctrl+d set default  esc cancel") + "\n")
	return sb.String()
}
