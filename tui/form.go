package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavitammiste/cssh/config"
)

// Form field indices
const (
	fieldName = iota
	fieldGroup
	fieldTags
	fieldHost
	fieldUser
	fieldPort
	fieldIdentity
	fieldCommand
	fieldExtraArgs
	fieldCount
)

// Port forward form field indices
const (
	pfFieldType = iota
	pfFieldLocalPort
	pfFieldRemoteHost
	pfFieldRemotePort
	pfFieldCount
)

type formModel struct {
	inputs       [fieldCount]textinput.Model
	pfInputs     [pfFieldCount]textinput.Model
	focused      int
	pfFocused    int
	portForwards []config.PortForward
	editingPF    bool
	isEdit       bool
	editGroup    string
	errMsg       string
}

func newFormModel(conn *config.Connection, groupName string) formModel {
	m := formModel{}

	placeholders := []string{"My Server", "Production", "production, web", "192.168.1.1", "ubuntu", "22", "~/.ssh/id_rsa", "", ""}

	for i := range m.inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.CharLimit = 256
		m.inputs[i] = t
	}

	// Initialize port forward inputs
	pfPlaceholders := []string{"L", "5432", "localhost", "5432"}
	for i := range m.pfInputs {
		t := textinput.New()
		t.Placeholder = pfPlaceholders[i]
		t.CharLimit = 64
		m.pfInputs[i] = t
	}

	if conn != nil {
		m.isEdit = true
		m.editGroup = groupName
		m.inputs[fieldName].SetValue(conn.Name)
		m.inputs[fieldGroup].SetValue(groupName)
		m.inputs[fieldTags].SetValue(strings.Join(conn.Tags, ", "))
		m.inputs[fieldHost].SetValue(conn.Host)
		m.inputs[fieldUser].SetValue(conn.User)
		if conn.Port > 0 {
			m.inputs[fieldPort].SetValue(fmt.Sprintf("%d", conn.Port))
		}
		m.inputs[fieldIdentity].SetValue(conn.IdentityFile)
		m.inputs[fieldCommand].SetValue(conn.Command)
		m.inputs[fieldExtraArgs].SetValue(conn.ExtraArgs)
		m.portForwards = make([]config.PortForward, len(conn.PortForwards))
		copy(m.portForwards, conn.PortForwards)
	}

	m.inputs[fieldName].Focus()
	return m
}

func (m *formModel) focusField(idx int) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	if idx >= 0 && idx < fieldCount {
		m.inputs[idx].Focus()
		m.focused = idx
	}
}

func (m *formModel) focusPFField(idx int) {
	for i := range m.pfInputs {
		m.pfInputs[i].Blur()
	}
	if idx >= 0 && idx < pfFieldCount {
		m.pfInputs[idx].Focus()
		m.pfFocused = idx
	}
}

func (m *formModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	if m.editingPF {
		m.pfInputs[m.pfFocused], cmd = m.pfInputs[m.pfFocused].Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *formModel) toConnection() (*config.Connection, string, error) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}
	group := strings.TrimSpace(m.inputs[fieldGroup].Value())
	if group == "" {
		group = "Default"
	}

	portStr := strings.TrimSpace(m.inputs[fieldPort].Value())
	port := 0
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, "", fmt.Errorf("invalid port: %s", portStr)
		}
		if p < 1 || p > 65535 {
			return nil, "", fmt.Errorf("port must be between 1 and 65535")
		}
		port = p
	}

	var tags []string
	for _, t := range strings.Split(m.inputs[fieldTags].Value(), ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	host := strings.TrimSpace(m.inputs[fieldHost].Value())
	command := strings.TrimSpace(m.inputs[fieldCommand].Value())

	if host == "" && command == "" {
		return nil, "", fmt.Errorf("host or command is required")
	}

	conn := &config.Connection{
		Name:         name,
		Tags:         tags,
		Host:         host,
		User:         strings.TrimSpace(m.inputs[fieldUser].Value()),
		Port:         port,
		IdentityFile: strings.TrimSpace(m.inputs[fieldIdentity].Value()),
		Command:      command,
		ExtraArgs:    strings.TrimSpace(m.inputs[fieldExtraArgs].Value()),
		PortForwards: m.portForwards,
	}

	return conn, group, nil
}

func (m *formModel) commitPortForward() error {
	typeVal := strings.ToUpper(strings.TrimSpace(m.pfInputs[pfFieldType].Value()))
	if typeVal != "L" && typeVal != "R" && typeVal != "D" {
		return fmt.Errorf("type must be L, R, or D")
	}

	localPortStr := strings.TrimSpace(m.pfInputs[pfFieldLocalPort].Value())
	localPort, err := strconv.Atoi(localPortStr)
	if err != nil || localPort <= 0 {
		return fmt.Errorf("invalid local port")
	}

	pf := config.PortForward{
		Type:      typeVal,
		LocalPort: localPort,
	}

	if typeVal != "D" {
		remoteHost := strings.TrimSpace(m.pfInputs[pfFieldRemoteHost].Value())
		remotePortStr := strings.TrimSpace(m.pfInputs[pfFieldRemotePort].Value())
		remotePort, err := strconv.Atoi(remotePortStr)
		if err != nil || remotePort <= 0 {
			return fmt.Errorf("invalid remote port")
		}
		pf.RemoteHost = remoteHost
		pf.RemotePort = remotePort
	}

	m.portForwards = append(m.portForwards, pf)
	return nil
}

func (m *formModel) clearPFInputs() {
	for i := range m.pfInputs {
		m.pfInputs[i].SetValue("")
		m.pfInputs[i].Blur()
	}
}

func (m formModel) viewForm(title string, width int) string {
	var sb strings.Builder

	sb.WriteString(formTitleStyle.Render(title) + "\n")
	sb.WriteString(dividerStyle.Render(strings.Repeat("─", min(width-4, 45))) + "\n")

	topFields := []struct {
		label string
		idx   int
	}{
		{"Name", fieldName},
		{"Group", fieldGroup},
		{"Tags", fieldTags},
	}

	for _, f := range topFields {
		label := formLabelStyle.Render(fmt.Sprintf("%-14s", f.label))
		if m.focused == f.idx {
			label = formLabelActiveStyle.Render(fmt.Sprintf("%-14s", f.label))
		}
		sb.WriteString(label + m.inputs[f.idx].View() + "\n")
		// Show rendered tag badges as preview
		if f.idx == fieldTags {
			preview := renderTagPreview(m.inputs[fieldTags].Value())
			if preview != "" {
				sb.WriteString(preview + "\n")
			}
		}
	}

	sb.WriteString(formSectionStyle.Render(dividerLine("Connection", width)) + "\n")

	connFields := []struct {
		label string
		idx   int
	}{
		{"Host", fieldHost},
		{"User", fieldUser},
		{"Port", fieldPort},
		{"Identity File", fieldIdentity},
	}

	for _, f := range connFields {
		label := formLabelStyle.Render(fmt.Sprintf("%-14s", f.label))
		if m.focused == f.idx {
			label = formLabelActiveStyle.Render(fmt.Sprintf("%-14s", f.label))
		}
		sb.WriteString(label + m.inputs[f.idx].View() + "\n")
		if f.idx == fieldIdentity && m.focused == fieldIdentity {
			sb.WriteString(helpStyle.Render("               ctrl+k browse ~/.ssh") + "\n")
		}
	}

	sb.WriteString(formSectionStyle.Render(dividerLine("Or Custom Command", width)) + "\n")

	cmdFields := []struct {
		label string
		idx   int
	}{
		{"Command", fieldCommand},
		{"Extra Args", fieldExtraArgs},
	}

	for _, f := range cmdFields {
		label := formLabelStyle.Render(fmt.Sprintf("%-14s", f.label))
		if m.focused == f.idx {
			label = formLabelActiveStyle.Render(fmt.Sprintf("%-14s", f.label))
		}
		sb.WriteString(label + m.inputs[f.idx].View() + "\n")
	}

	sb.WriteString(formSectionStyle.Render(dividerLine("Port Forwards", width)) + "\n")

	for _, pf := range m.portForwards {
		var pfStr string
		switch pf.Type {
		case "D":
			pfStr = fmt.Sprintf("D:%d", pf.LocalPort)
		default:
			pfStr = fmt.Sprintf("%s:%d → %s:%d", pf.Type, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
		}
		sb.WriteString(pfItemStyle.Render("  • "+pfStr) + "\n")
	}

	sb.WriteString(helpStyle.Render("  ctrl+f add forward  ctrl+r remove last") + "\n")

	sb.WriteString(dividerStyle.Render(strings.Repeat("─", min(width-4, 45))) + "\n")
	sb.WriteString(helpStyle.Render("tab/↑↓ navigate  ctrl+s save  esc cancel") + "\n")

	if m.errMsg != "" {
		sb.WriteString("\n" + errorStyle.Render("  "+m.errMsg) + "\n")
	}

	return sb.String()
}

func (m formModel) viewPortForwardForm(width int) string {
	var sb strings.Builder

	sb.WriteString(formTitleStyle.Render("Add Port Forward") + "\n")
	sb.WriteString(dividerStyle.Render(strings.Repeat("─", min(width-4, 45))) + "\n")

	pfFields := []struct {
		label string
		idx   int
	}{
		{"Type (L/R/D)", pfFieldType},
		{"Local Port", pfFieldLocalPort},
		{"Remote Host", pfFieldRemoteHost},
		{"Remote Port", pfFieldRemotePort},
	}

	typeVal := strings.ToUpper(strings.TrimSpace(m.pfInputs[pfFieldType].Value()))
	isDynamic := typeVal == "D"

	for _, f := range pfFields {
		if isDynamic && (f.idx == pfFieldRemoteHost || f.idx == pfFieldRemotePort) {
			continue
		}
		label := formLabelStyle.Render(fmt.Sprintf("%-14s", f.label))
		if m.pfFocused == f.idx {
			label = formLabelActiveStyle.Render(fmt.Sprintf("%-14s", f.label))
		}
		sb.WriteString(label + m.pfInputs[f.idx].View() + "\n")
	}

	sb.WriteString(dividerStyle.Render(strings.Repeat("─", min(width-4, 45))) + "\n")
	sb.WriteString(helpStyle.Render("tab navigate  enter confirm  esc cancel") + "\n")

	if m.errMsg != "" {
		sb.WriteString("\n" + errorStyle.Render("  "+m.errMsg) + "\n")
	}

	return sb.String()
}

// renderTagPreview shows comma-separated tag values as colored badges.
func renderTagPreview(raw string) string {
	var parts []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			parts = append(parts, tagStyle(t).Render(t))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "               " + strings.Join(parts, " ")
}

func dividerLine(label string, width int) string {
	maxW := min(width-4, 45)
	dashes := maxW - len(label) - 4
	if dashes < 2 {
		dashes = 2
	}
	return fmt.Sprintf("─── %s %s", label, strings.Repeat("─", dashes))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
