package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds the full color palette for a UI theme.
type Theme struct {
	Name   string
	Blue   lipgloss.Color // primary accent: title, selection, cursor, active inputs
	Purple lipgloss.Color
	Cyan   lipgloss.Color
	White  lipgloss.Color
	Gray   lipgloss.Color
	Yellow lipgloss.Color
	SelBg  lipgloss.Color
	Border lipgloss.Color
	Red    lipgloss.Color
	Green  lipgloss.Color
	Orange lipgloss.Color
}

var Themes = []Theme{
	{
		Name:   "Tokyo Night",
		Blue:   "#7AA2F7",
		Purple: "#BB9AF7",
		Cyan:   "#7DCFFF",
		White:  "#C0CAF5",
		Gray:   "#737AA2",
		Yellow: "#E0AF68",
		SelBg:  "#283457",
		Border: "#414868",
		Red:    "#F7768E",
		Green:  "#9ECE6A",
		Orange: "#FF9E64",
	},
	{
		Name:   "Catppuccin Mocha",
		Blue:   "#89B4FA",
		Purple: "#CBA6F7",
		Cyan:   "#89DCEB",
		White:  "#CDD6F4",
		Gray:   "#6C7086",
		Yellow: "#F9E2AF",
		SelBg:  "#313244",
		Border: "#45475A",
		Red:    "#F38BA8",
		Green:  "#A6E3A1",
		Orange: "#FAB387",
	},
	{
		Name:   "Gruvbox Dark",
		Blue:   "#83A598",
		Purple: "#D3869B",
		Cyan:   "#8EC07C",
		White:  "#EBDBB2",
		Gray:   "#928374",
		Yellow: "#FABD2F",
		SelBg:  "#3C3836",
		Border: "#665C54",
		Red:    "#FB4934",
		Green:  "#B8BB26",
		Orange: "#FE8019",
	},
	{
		Name:   "Linux Console",
		Blue:   "#5F87AF",
		Purple: "#87AF87",
		Cyan:   "#87D7AF",
		White:  "#D0D0D0",
		Gray:   "#808080",
		Yellow: "#FFD75F",
		SelBg:  "#262626",
		Border: "#5F5F5F",
		Red:    "#D75F5F",
		Green:  "#87AF5F",
		Orange: "#FFAF5F",
	},
}

var currentThemeIdx = 0

func CurrentTheme() Theme { return Themes[currentThemeIdx] }

func SetThemeByName(name string) {
	for i, t := range Themes {
		if t.Name == name {
			currentThemeIdx = i
			ApplyTheme(t)
			return
		}
	}
}

// NextTheme cycles to the next theme, applies it, and returns it.
func NextTheme() Theme {
	currentThemeIdx = (currentThemeIdx + 1) % len(Themes)
	t := Themes[currentThemeIdx]
	ApplyTheme(t)
	return t
}
