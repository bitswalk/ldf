package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

var (
	logger *log.Logger

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))
)

// Model represents the application state
type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
	quitting bool
}

func initialModel() model {
	return model{
		choices: []string{
			"Create New Distribution",
			"List Distributions",
			"Build Distribution",
			"Manage Board Profiles",
			"Settings",
			"Exit",
		},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			if m.cursor == len(m.choices)-1 {
				// Exit option
				m.quitting = true
				return m, tea.Quit
			}
			// TODO: Handle other menu selections
			logger.Info("Selected", "option", m.choices[m.cursor])
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Thanks for using Linux Distribution Factory!\n"
	}

	s := titleStyle.Render("🐧 Linux Distribution Factory") + "\n\n"
	s += "Welcome! Use arrow keys to navigate, Enter to select, and 'q' to quit.\n\n"

	// Render menu items
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
			s += cursor + selectedItemStyle.Render(choice) + "\n"
		} else {
			s += cursor + normalItemStyle.Render(choice) + "\n"
		}
	}

	s += "\n" + lipgloss.NewStyle().Faint(true).Render("Press 'q' to quit")

	return s
}

func init() {
	// Initialize logger
	logger = log.New(os.Stderr)

	// Set up configuration
	viper.SetDefault("log.level", "info")
	viper.SetDefault("tui.theme", "default")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/ldf/")
	viper.AddConfigPath("$HOME/.config/ldf")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("LDF")

	if err := viper.ReadInConfig(); err != nil {
		logger.Debug("No config file found, using defaults")
	} else {
		logger.Info("Using config file", "file", viper.ConfigFileUsed())
	}

	// Set log level
	switch viper.GetString("log.level") {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.InfoLevel)
	}
}

func main() {
	// Initialize the TUI program
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		logger.Fatal("Error running TUI", "error", err)
		os.Exit(1)
	}
}
