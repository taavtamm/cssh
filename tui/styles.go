package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle           lipgloss.Style
	dividerStyle         lipgloss.Style
	groupHeaderStyle     lipgloss.Style
	connNameStyle        lipgloss.Style
	connHostStyle        lipgloss.Style
	selectedStyle        lipgloss.Style
	selectedNameStyle    lipgloss.Style
	selectedHostStyle    lipgloss.Style
	badgeStyle           lipgloss.Style
	helpStyle            lipgloss.Style
	errorStyle           lipgloss.Style
	successStyle         lipgloss.Style
	formTitleStyle       lipgloss.Style
	formSectionStyle     lipgloss.Style
	formLabelStyle       lipgloss.Style
	formLabelActiveStyle lipgloss.Style
	confirmStyle         lipgloss.Style
	markerStyle          lipgloss.Style
	pfItemStyle          lipgloss.Style
	borderStyle          lipgloss.Style
	searchActiveStyle    lipgloss.Style
	themeNameStyle       lipgloss.Style

	tagColorPalette []lipgloss.Color
	tagForeground   lipgloss.Color
)

func init() { ApplyTheme(Themes[0]) }

func ApplyTheme(t Theme) {
	tagForeground = lipgloss.Color("#1A1B2E")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Blue)
	dividerStyle = lipgloss.NewStyle().Foreground(t.Border)
	groupHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Purple)
	connNameStyle = lipgloss.NewStyle().Foreground(t.White)
	connHostStyle = lipgloss.NewStyle().Foreground(t.Gray)
	selectedStyle = lipgloss.NewStyle().Background(t.SelBg).Foreground(t.Blue)
	selectedNameStyle = lipgloss.NewStyle().Background(t.SelBg).Foreground(t.Blue).Bold(true)
	selectedHostStyle = lipgloss.NewStyle().Background(t.SelBg).Foreground(t.White)
	badgeStyle = lipgloss.NewStyle().Foreground(t.Yellow)
	helpStyle = lipgloss.NewStyle().Foreground(t.Gray)
	errorStyle = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(t.Green)
	formTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Blue)
	formSectionStyle = lipgloss.NewStyle().Foreground(t.Gray).Bold(true)
	formLabelStyle = lipgloss.NewStyle().Foreground(t.Gray)
	formLabelActiveStyle = lipgloss.NewStyle().Foreground(t.Blue)
	confirmStyle = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	markerStyle = lipgloss.NewStyle().Foreground(t.Blue).Bold(true)
	pfItemStyle = lipgloss.NewStyle().Foreground(t.Yellow)
	borderStyle = lipgloss.NewStyle().Foreground(t.Border)
	searchActiveStyle = lipgloss.NewStyle().Foreground(t.Blue)
	themeNameStyle = lipgloss.NewStyle().Foreground(t.Gray)

	tagColorPalette = []lipgloss.Color{
		t.Red, t.Yellow, t.Green, t.Cyan,
		t.Purple, t.Orange, "#73DACA", "#9D7CD8",
	}
}

// tagStyle returns a background-colored badge style for a tag.
// Common tags get specific semantic colors; others use a hash-picked palette color.
func tagStyle(tag string) lipgloss.Style {
	special := map[string]lipgloss.Color{
		"production":  "#F7768E",
		"prod":        "#F7768E",
		"staging":     "#E0AF68",
		"stage":       "#E0AF68",
		"deprecated":  "#565F89",
		"dev":         "#7DCFFF",
		"development": "#7DCFFF",
		"critical":    "#FF5555",
		"backup":      "#9ECE6A",
	}
	if c, ok := special[strings.ToLower(tag)]; ok {
		return lipgloss.NewStyle().
			Background(c).
			Foreground(tagForeground).
			Padding(0, 1).
			Bold(true)
	}
	h := 0
	for _, r := range tag {
		h = h*31 + int(r)
	}
	if h < 0 {
		h = -h
	}
	color := tagColorPalette[h%len(tagColorPalette)]
	return lipgloss.NewStyle().
		Background(color).
		Foreground(tagForeground).
		Padding(0, 1).
		Bold(true)
}
