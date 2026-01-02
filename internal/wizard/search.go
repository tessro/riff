package wizard

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchType represents the type of search to perform.
type SearchType int

const (
	SearchAll SearchType = iota
	SearchTracks
	SearchAlbums
	SearchArtists
	SearchPlaylists
)

// SearchResult represents a search result item.
type SearchResult struct {
	ID       string
	URI      string
	Title    string
	Subtitle string
	Type     SearchType
}

// SearchFunc is a function that performs a search.
type SearchFunc func(query string, searchType SearchType) ([]SearchResult, error)

// SearchModel is the bubbletea model for the search wizard.
type SearchModel struct {
	input       textinput.Model
	results     []SearchResult
	cursor      int
	searchType  SearchType
	searchFunc  SearchFunc
	selected    *SearchResult
	err         error
	debounce    time.Duration
	lastQuery   string
	searching   bool
	width       int
	height      int
}

// Styles
var (
	searchTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	searchTabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	searchActiveTabStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Background(lipgloss.Color("205")).
				Foreground(lipgloss.Color("0"))

	searchResultStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	searchSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Background(lipgloss.Color("237"))

	searchSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))
)

// NewSearchModel creates a new search wizard model.
func NewSearchModel(searchFunc SearchFunc) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search for tracks, albums, artists..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return SearchModel{
		input:      ti,
		searchFunc: searchFunc,
		debounce:   300 * time.Millisecond,
		searchType: SearchAll,
		width:      80,
		height:     20,
	}
}

// Init initializes the model.
func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// debounceMsg is sent after the debounce period.
type debounceMsg struct {
	query string
}

// searchResultsMsg contains search results.
type searchResultsMsg struct {
	results []SearchResult
	err     error
}

// Update handles messages.
func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				m.selected = &m.results[m.cursor]
				return m, tea.Quit
			}

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "ctrl+n":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}

		case "tab":
			// Cycle through search types
			m.searchType = (m.searchType + 1) % 5
			if m.input.Value() != "" {
				return m, m.doSearch(m.input.Value())
			}

		case "shift+tab":
			// Cycle backwards
			if m.searchType == 0 {
				m.searchType = 4
			} else {
				m.searchType--
			}
			if m.input.Value() != "" {
				return m, m.doSearch(m.input.Value())
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4

	case debounceMsg:
		if msg.query == m.input.Value() && msg.query != m.lastQuery {
			m.lastQuery = msg.query
			return m, m.doSearch(msg.query)
		}

	case searchResultsMsg:
		m.searching = false
		m.results = msg.results
		m.err = msg.err
		m.cursor = 0
	}

	// Handle text input
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

	// Debounce search
	if m.input.Value() != m.lastQuery {
		cmds = append(cmds, tea.Tick(m.debounce, func(time.Time) tea.Msg {
			return debounceMsg{query: m.input.Value()}
		}))
	}

	return m, tea.Batch(cmds...)
}

// doSearch performs the search.
func (m SearchModel) doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return searchResultsMsg{results: nil}
		}
		results, err := m.searchFunc(query, m.searchType)
		return searchResultsMsg{results: results, err: err}
	}
}

// View renders the model.
func (m SearchModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(searchTitleStyle.Render("🔍 Search"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	// Type filter tabs
	tabs := []string{"All", "Tracks", "Albums", "Artists", "Playlists"}
	for i, tab := range tabs {
		if SearchType(i) == m.searchType {
			b.WriteString(searchActiveTabStyle.Render(tab))
		} else {
			b.WriteString(searchTabStyle.Render(tab))
		}
	}
	b.WriteString("\n\n")

	// Results
	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error()))
	} else if m.searching {
		b.WriteString("Searching...")
	} else if len(m.results) == 0 && m.input.Value() != "" {
		b.WriteString("No results found")
	} else {
		maxResults := m.height - 10
		if maxResults < 5 {
			maxResults = 5
		}
		for i, result := range m.results {
			if i >= maxResults {
				b.WriteString(searchSubtitleStyle.Render("  ...and more"))
				break
			}

			line := result.Title
			if result.Subtitle != "" {
				line += " " + searchSubtitleStyle.Render(result.Subtitle)
			}

			if i == m.cursor {
				b.WriteString(searchSelectedStyle.Render("▸ " + line))
			} else {
				b.WriteString(searchResultStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(searchSubtitleStyle.Render("↑/↓ navigate • tab switch type • enter select • esc quit"))

	return b.String()
}

// Selected returns the selected result, or nil if none.
func (m SearchModel) Selected() *SearchResult {
	return m.selected
}

// RunSearch runs the search wizard and returns the selected result.
func RunSearch(searchFunc SearchFunc) (*SearchResult, error) {
	model := NewSearchModel(searchFunc)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(SearchModel).Selected(), nil
}
